package netbatch

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// testEntry is one entry a test saw, with its payload.
type testEntry struct {
	handle  int32
	trigger NetTrigger
	detail  int32
	peer    int32
	payload string
}

// testUpdate runs one update and returns its entries.
func testUpdate(t *testing.T, waitMilliseconds int32) []testEntry {
	t.Helper()
	count := NetUpdate(waitMilliseconds)
	if count < 0 {
		t.Fatal("the replay record has no more updates")
	}
	if NetReadyCount() != count {
		t.Fatalf("ready count %d after an update of %d entries", NetReadyCount(), count)
	}
	var entries []testEntry
	var buffer [NetRingSize]byte
	for {
		handle, trigger, messageID, detail := NetReadyNext()
		if trigger == NetTriggerNone {
			break
		}
		entry := testEntry{handle: handle, trigger: trigger, detail: detail}
		if trigger == NetTriggerData {
			length, peer := NetUDPReadFrom(handle, messageID, buffer[:], 0)
			if peer == 0 {
				length = NetRead(handle, messageID, buffer[:], 0)
			}
			if length != detail {
				t.Fatalf("read %d bytes of a %d byte entry", length, detail)
			}
			entry.payload, entry.peer = string(buffer[:length]), peer
		}
		entries = append(entries, entry)
	}
	if len(entries) != int(count) {
		t.Fatalf("%d entries of an update of %d", len(entries), count)
	}
	return entries
}

// testUpdateUntil updates until done reports true of the entries so far, or
// fails after five seconds.
func testUpdateUntil(t *testing.T, done func([]testEntry) bool) []testEntry {
	t.Helper()
	var entries []testEntry
	deadline := time.Now().Add(5 * time.Second)
	for !done(entries) {
		if time.Now().After(deadline) {
			t.Fatalf("no wanted entry in five seconds, entries %+v", entries)
		}
		entries = append(entries, testUpdate(t, 20)...)
	}
	return entries
}

// testFind returns the first entry of handle with trigger.
func testFind(entries []testEntry, handle int32, trigger NetTrigger) (testEntry, bool) {
	for _, entry := range entries {
		if entry.handle == handle && entry.trigger == trigger {
			return entry, true
		}
	}
	return testEntry{}, false
}

// testPayload joins the data of handle.
func testPayload(entries []testEntry, handle int32) string {
	var payload string
	for _, entry := range entries {
		if entry.handle == handle && entry.trigger == NetTriggerData {
			payload += entry.payload
		}
	}
	return payload
}

func testRunStart(t *testing.T, mode NetMode, path string) {
	t.Helper()
	if err := NetRunStart(mode, path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(NetRunStop)
}

// testListen opens a TCP listener on a loopback port and returns its handle
// and port.
func testListen(t *testing.T) (int32, int32) {
	t.Helper()
	listener := NetTCPListen("127.0.0.1:0")
	entries := testUpdate(t, 0)
	listening, found := testFind(entries, listener, NetTriggerListening)
	if !found {
		t.Fatalf("no listening entry: %+v", entries)
	}
	return listener, listening.detail
}

// testEcho is a client and server on loopback. The client sends hello, the
// server answers world and closes.
func testEcho(t *testing.T) []testEntry {
	listener, port := testListen(t)
	client := NetTCPDial("127.0.0.1:" + strconv.Itoa(int(port)))
	var all []testEntry
	var server int32
	all = append(all, testUpdateUntil(t, func(entries []testEntry) bool {
		accepted, acceptedFound := testFind(entries, listener, NetTriggerAccepted)
		_, connectedFound := testFind(entries, client, NetTriggerConnected)
		server = accepted.detail
		return acceptedFound && connectedFound
	})...)
	if NetWrite(client, []byte("hello")) != 5 {
		t.Fatal("write of the client")
	}
	all = append(all, testUpdateUntil(t, func(entries []testEntry) bool {
		return testPayload(entries, server) == "hello"
	})...)
	NetWrite(server, []byte("world"))
	NetClose(server)
	all = append(all, testUpdateUntil(t, func(entries []testEntry) bool {
		_, closed := testFind(entries, client, NetTriggerClosed)
		return testPayload(entries, client) == "world" && closed
	})...)
	NetClose(client)
	NetClose(listener)
	return all
}

func TestNetTCPEcho(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	testEcho(t)
}

func TestNetWriteBeforeConnected(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	_, port := testListen(t)
	client := NetTCPDial("127.0.0.1:" + strconv.Itoa(int(port)))
	if queued := NetWrite(client, []byte("early")); queued != 0 {
		t.Fatalf("write before the connected entry queued %d bytes", queued)
	}
}

func TestNetDialRefused(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	listener, port := testListen(t)
	NetClose(listener)
	client := NetTCPDial("127.0.0.1:" + strconv.Itoa(int(port)))
	entries := testUpdateUntil(t, func(entries []testEntry) bool {
		_, found := testFind(entries, client, NetTriggerError)
		return found
	})
	failed, _ := testFind(entries, client, NetTriggerError)
	if NetError(failed.detail) != NetErrorRefused {
		t.Fatalf("dial of a closed port: error %d, want %d", failed.detail, NetErrorRefused)
	}
}

func TestNetUDP(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	server := NetUDPListen("127.0.0.1:0")
	listening, found := testFind(testUpdate(t, 0), server, NetTriggerListening)
	if !found {
		t.Fatal("no listening entry")
	}
	client := NetUDPDial("127.0.0.1:" + strconv.Itoa(int(listening.detail)))
	if _, found := testFind(testUpdate(t, 0), client, NetTriggerConnected); !found {
		t.Fatal("no connected entry")
	}
	NetWrite(client, []byte("ping"))
	entries := testUpdateUntil(t, func(entries []testEntry) bool {
		return testPayload(entries, server) == "ping"
	})
	ping, _ := testFind(entries, server, NetTriggerData)
	if ping.peer != 1 {
		t.Fatalf("first peer is %d, want 1", ping.peer)
	}
	if NetUDPWriteTo(server, ping.peer, []byte("pong")) != 4 {
		t.Fatal("write to the peer")
	}
	testUpdateUntil(t, func(entries []testEntry) bool {
		return testPayload(entries, client) == "pong"
	})
	if peer := NetUDPPeerResolve("not an address"); peer != 0 {
		t.Fatalf("peer of a bad address is %d", peer)
	}
}

// A UDP socket that receives more than its ring holds between two updates
// drops the rest and says how many in a dropped entry.
func TestNetUDPDropped(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	server := NetUDPListen("127.0.0.1:0")
	listening, _ := testFind(testUpdate(t, 0), server, NetTriggerListening)
	sender, err := net.Dial("udp", "127.0.0.1:"+strconv.Itoa(int(listening.detail)))
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	datagram := make([]byte, 1000)
	const sent = 40
	for i := 0; i < sent; i++ {
		sender.Write(datagram)
		time.Sleep(time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	entries := testUpdate(t, 0)
	received, dropped := 0, 0
	for _, entry := range entries {
		switch entry.trigger {
		case NetTriggerData:
			received++
		case NetTriggerDropped:
			dropped += int(entry.detail)
		}
	}
	if received != NetRingSize/len(datagram) || dropped == 0 || received+dropped > sent {
		t.Fatalf("received %d, dropped %d of %d datagrams of %d bytes into a %d byte ring", received, dropped, sent, len(datagram), NetRingSize)
	}
}

// A TCP reader stops when the ring is full and the peer's writes wait, and
// every byte arrives once the program updates.
func TestNetTCPBackpressure(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	listener, port := testListen(t)
	peer, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(int(port)))
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	accepted, _ := testFind(testUpdateUntil(t, func(entries []testEntry) bool {
		_, found := testFind(entries, listener, NetTriggerAccepted)
		return found
	}), listener, NetTriggerAccepted)
	const total = 8 << 20
	written := make(chan error, 1)
	go func() {
		_, err := peer.Write(bytes.Repeat([]byte{'x'}, total))
		written <- err
	}()
	time.Sleep(100 * time.Millisecond)
	select {
	case <-written:
		t.Fatal("8 MiB were written to a program that does not update")
	default:
	}
	received := 0
	deadline := time.Now().Add(20 * time.Second)
	for received < total && time.Now().Before(deadline) {
		for _, entry := range testUpdate(t, 20) {
			if entry.handle == accepted.detail && entry.trigger == NetTriggerData {
				received += len(entry.payload)
			}
		}
	}
	if received != total || <-written != nil {
		t.Fatalf("received %d of %d bytes", received, total)
	}
}

func TestNetTimeout(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	listener, _ := testListen(t)
	NetTimeoutSet(listener, 30)
	start := NetClock()
	entries := testUpdateUntil(t, func(entries []testEntry) bool {
		_, found := testFind(entries, listener, NetTriggerTimeout)
		return found
	})
	if elapsed := NetClock() - start; elapsed < 30 {
		t.Fatalf("timeout of 30 ms fired after %d ms: %+v", elapsed, entries)
	}
	NetTimeoutSet(listener, 10)
	NetTimeoutSet(listener, 0)
	time.Sleep(20 * time.Millisecond)
	if _, found := testFind(testUpdate(t, 0), listener, NetTriggerTimeout); found {
		t.Fatal("a stopped timeout fired")
	}
}

func TestNetTimerWheel(t *testing.T) {
	var wheel NetTimerWheel
	wheel.timerWheelClear()
	var fired []int32
	fire := func(handle int32) { fired = append(fired, handle) }
	wheel.TimerStart(1, 5)
	wheel.TimerStart(2, 5+netTimerSlotCount)
	wheel.TimerStart(3, 7)
	wheel.TimerStop(3)
	wheel.TimerAdvance(5, fire)
	if fmt.Sprint(fired) != "[1]" {
		t.Fatalf("at 5: fired %v", fired)
	}
	wheel.TimerAdvance(10*netTimerSlotCount, fire)
	if fmt.Sprint(fired) != "[1 2]" {
		t.Fatalf("after a long jump: fired %v", fired)
	}
}

// A recorded run replays to the same entries without a socket.
func TestNetRecordReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "echo.netreplay")
	testRunStart(t, NetModeRecord, path)
	live := testEcho(t)
	updates := netState.netUpdateNumber
	NetRunStop()

	testRunStart(t, NetModeReplay, path)
	if NetReplayUpdatesLeft() != int(updates) {
		t.Fatalf("record holds %d updates, the run made %d", NetReplayUpdatesLeft(), updates)
	}
	var replayed []testEntry
	listener := NetTCPListen("127.0.0.1:0")
	replayed = append(replayed, testUpdate(t, 0)...)
	listening, _ := testFind(replayed, listener, NetTriggerListening)
	replayed = nil
	client := NetTCPDial("127.0.0.1:" + strconv.Itoa(int(listening.detail)))
	for NetReplayUpdatesLeft() > 0 {
		entries := testUpdate(t, 0)
		replayed = append(replayed, entries...)
		for _, entry := range entries {
			if entry.trigger == NetTriggerData && entry.handle != client {
				NetWrite(entry.handle, []byte("world"))
				NetClose(entry.handle)
			}
		}
	}
	if fmt.Sprint(live) != fmt.Sprint(replayed) {
		t.Fatalf("replay differs\nlive     %+v\nreplayed %+v", live, replayed)
	}
	if NetUpdate(0) != -1 {
		t.Fatal("an update past the end of the record")
	}
}

func TestNetPayloadQuote(t *testing.T) {
	payload := []byte("a \"b\\\n\x00\xff")
	quoted := string(netPayloadQuote(nil, payload))
	unquoted, err := strconv.Unquote(quoted)
	if err != nil || unquoted != string(payload) {
		t.Fatalf("%s unquotes to %q, %v", quoted, unquoted, err)
	}
}

// testPairs dials count connections to a plain Go listener and returns the
// accepted peers by handle.
func testPairs(t testing.TB, count int) map[int32]net.Conn {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, count)
	go func() {
		for i := 0; i < count; i++ {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			accepted <- conn
		}
	}()
	address := listener.Addr().String()
	handles := make([]int32, count)
	for i := range handles {
		handles[i] = NetTCPDial(address)
	}
	connected := 0
	deadline := time.Now().Add(20 * time.Second)
	for connected < count && time.Now().Before(deadline) {
		NetUpdate(20)
		for {
			_, trigger, _, _ := NetReadyNext()
			if trigger == NetTriggerNone {
				break
			}
			if trigger == NetTriggerConnected {
				connected++
			}
		}
	}
	if connected != count {
		t.Fatalf("%d of %d connected", connected, count)
	}
	// The peers are matched to handles by a byte each handle sends.
	peers := map[int32]net.Conn{}
	for _, handle := range handles {
		NetWrite(handle, []byte{byte(handle), byte(handle >> 8)})
	}
	NetUpdate(0)
	for i := 0; i < count; i++ {
		conn := <-accepted
		var id [2]byte
		if _, err := io.ReadFull(conn, id[:]); err != nil {
			t.Fatal(err)
		}
		peers[int32(id[0])|int32(id[1])<<8] = conn
	}
	return peers
}

// testReadyWait waits until ready arrivals are in the channel.
func testReadyWait(ready int) {
	deadline := time.Now().Add(5 * time.Second)
	for len(netState.netArrivals) < ready && time.Now().Before(deadline) {
		time.Sleep(50 * time.Microsecond)
	}
}

// The work of an update grows with the ready count and not with the number
// of open sockets (#508).
func TestNetUpdateWorkGrowsWithReady(t *testing.T) {
	work := map[[2]int]int64{}
	for _, count := range []int{16, 512} {
		testRunStart(t, NetModeLive, "")
		peers := testPairs(t, count)
		for _, ready := range []int{1, 4, 16} {
			for handle, peer := range peers {
				if handle <= int32(ready) {
					peer.Write([]byte{1})
				}
			}
			testReadyWait(ready)
			before := netState.NetUpdateWork
			NetUpdate(0)
			NetUpdate(0)
			work[[2]int{count, ready}] = netState.NetUpdateWork - before
		}
		for _, peer := range peers {
			peer.Close()
		}
		NetRunStop()
	}
	for _, ready := range []int{1, 4, 16} {
		if work[[2]int{16, ready}] != work[[2]int{512, ready}] {
			t.Errorf("ready %d: work %d with 16 sockets, %d with 512", ready, work[[2]int{16, ready}], work[[2]int{512, ready}])
		}
	}
	if work[[2]int{512, 16}] <= work[[2]int{512, 1}] {
		t.Errorf("work does not grow with the ready count: %v", work)
	}
	t.Logf("work by sockets and ready count: %v", work)
}

// An update, the ready reads and a read make no Go allocation.
func TestNetUpdateAllocs(t *testing.T) {
	testRunStart(t, NetModeLive, "")
	peers := testPairs(t, 4)
	var buffer [64]byte
	payload := []byte("ab")
	allocs := testing.AllocsPerRun(50, func() {
		for _, peer := range peers {
			peer.Write(payload)
		}
		testReadyWait(len(peers))
		NetUpdate(0)
		for {
			handle, trigger, messageID, _ := NetReadyNext()
			if trigger == NetTriggerNone {
				break
			}
			NetRead(handle, messageID, buffer[:], 0)
			NetWrite(handle, buffer[:1])
		}
		NetReadyCount()
		NetClock()
	})
	if allocs != 0 {
		t.Fatalf("%.1f allocations per update", allocs)
	}
}

// BenchmarkNetUpdate times one update with ready of sockets open having data.
func BenchmarkNetUpdate(b *testing.B) {
	for _, count := range []int{16, 256, 1000} {
		for _, ready := range []int{1, 16, 64} {
			if ready > count {
				continue
			}
			b.Run(fmt.Sprintf("sockets=%d/ready=%d", count, ready), func(b *testing.B) {
				if err := NetRunStart(NetModeLive, ""); err != nil {
					b.Fatal(err)
				}
				defer NetRunStop()
				peers := testPairs(b, count)
				var writers []net.Conn
				for handle, peer := range peers {
					if handle <= int32(ready) {
						writers = append(writers, peer)
					}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					for _, peer := range writers {
						peer.Write([]byte{1})
					}
					testReadyWait(ready)
					b.StartTimer()
					NetUpdate(0)
				}
				b.StopTimer()
				for _, peer := range peers {
					peer.Close()
				}
			})
		}
	}
}
