# Networking

`sys/net` is the system library `net`, number 2, so a program imports it and
the run enables it.

```
cx --enable=net program.cx
```

A program never reads a socket. The runtime reads every socket in the
background and buffers what arrives. The program calls `net.Update`, which
makes everything that arrived since the last update one fixed batch, and then
reads the entries of the batch. Nothing the program can see changes between
two updates. See [skycoin-cx#508](https://github.com/skycoin/skycoin-cx/issues/508)
and the System libraries section of [LanguageGuide.md](../LanguageGuide.md#the-net-library).

## Functions

Every function is called through `LibraryCall` (795) with library 2 and its
function number, fixed for life in `constants.LibraryFunctionLifecycleTable`.
Numbers 1 to 4, the blocking `tcp.Dial`, `tcp.Listen`, `tcp.Accept` and
`tcp.Close`, are deprecated since 2611240004 and reassignable from
2711240001.

| Number | Function | Does |
| --- | --- | --- |
| 5 | `net.Update(waitMilliseconds i32) (count i32)` | makes the new batch and returns its entry count. When nothing has arrived it waits up to `waitMilliseconds` for something to arrive, 0 does not wait |
| 6 | `net.ReadyCount() (count i32)` | the entries of the batch `net.ReadyNext` has not returned yet |
| 7 | `net.ReadyNext() (handle i32, trigger i32, messageId i32, detail i32)` | the next entry, or `net.TriggerNone` after the last |
| 8 | `net.Clock() (milliseconds i64)` | the clock reading of the last update, milliseconds since the run started |
| 9 | `net.Read(handle i32, messageId i32, buffer [N]ui8, offset i32) (length i32)` | copies the payload of a data entry from byte `offset` on into `buffer` and returns the bytes copied |
| 10 | `net.Write(handle i32, buffer [N]ui8, length i32) (queued i32)` | queues the first `length` bytes of `buffer` for the next update and returns the bytes queued |
| 11 | `net.Close(handle i32) (closed bool)` | closes and frees the handle. What was written since the last update is still sent |
| 12 | `net.TimeoutSet(handle i32, milliseconds i32) (set bool)` | starts a timeout on the handle, 0 stops it |
| 13 | `net.TCPListen(host str, port i32) (handle i32)` | opens a TCP listener. Port 0 takes a free port |
| 14 | `net.TCPDial(host str, port i32) (handle i32)` | starts a TCP connection |
| 15 | `net.UDPListen(host str, port i32) (handle i32)` | opens a UDP socket bound to the address |
| 16 | `net.UDPDial(host str, port i32) (handle i32)` | opens a UDP socket whose `net.Write` goes to the address |
| 17 | `net.UDPReadFrom(handle i32, messageId i32, buffer [N]ui8, offset i32) (length i32, peer i32)` | `net.Read` for UDP, and the peer the datagram came from |
| 18 | `net.UDPWriteTo(handle i32, peer i32, buffer [N]ui8, length i32) (queued i32)` | queues one datagram to the peer, and returns its length or 0 when it does not fit |
| 19 | `net.UDPPeerResolve(host str, port i32) (peer i32)` | the peer of a numeric address, added to the peer table |

A function that opens a socket returns its handle at once, or 0 when the
handle table is full. Its result is an entry of a later update: listening,
connected or error. A function given a handle that is not open returns 0 or
false. A buffer is a `ui8` array of any length, a global or a local, read or
filled in place. Any other buffer is an assert on the old loop and does not
lower on vm.

## Entries

An entry is a socket handle, a trigger, a message id and a detail. The
message id is the index of the entry in the batch, which `net.Read` takes.

| Trigger | Value | Detail |
| --- | --- | --- |
| `net.TriggerNone` | 0 | after the last entry |
| `net.TriggerData` | 1 | the payload length |
| `net.TriggerAccepted` | 2 | the handle of the connection the listener accepted |
| `net.TriggerConnected` | 3 | 0. A TCP dial connected, or a UDP dial is ready |
| `net.TriggerListening` | 4 | the port the listener is bound to |
| `net.TriggerClosed` | 5 | 0. The peer closed the TCP connection |
| `net.TriggerError` | 6 | the error, below |
| `net.TriggerTimeout` | 7 | 0. A timeout `net.TimeoutSet` started expired |
| `net.TriggerDropped` | 8 | the UDP datagrams dropped since the last dropped entry of the socket |

| Error | Value |
| --- | --- |
| `net.ErrorOther` | 1 |
| `net.ErrorRefused` | 2 |
| `net.ErrorReset` | 3 |
| `net.ErrorTimeout` | 4, a TCP dial that did not connect in 30 seconds |
| `net.ErrorAddress` | 5, an address that does not parse or resolve |
| `net.ErrorAddressInUse` | 6 |
| `net.ErrorSocketTableFull` | 7, an accepted connection that found no free handle and was closed |

```cx
package main

import "sys/net"

var buffer [1024]ui8

func main() { global buffer
	var listener i32 = net.TCPListen("127.0.0.1", 9000)
	var count i32
	var handle i32
	var trigger i32
	var message i32
	var detail i32
	var length i32
	var queued i32
	var i i32
	for true {
		count = net.Update(1000)
		for i = 0; i < count; i++ {
			handle, trigger, message, detail = net.ReadyNext()
			if trigger == net.TriggerData {
				length = net.Read(handle, message, buffer, 0)
				queued = net.Write(handle, buffer, length) // echo
			}
		}
	}
}
```

## The update

`net.Update` does these steps, in order.

1. Frees the receive ring bytes of the batch before, so the readers waiting
   for space go on.
2. Hands what each socket was written since the last update to its writer.
   It waits for a writer still sending the update before, which only a
   peer that stopped reading makes slow.
3. Waits up to `waitMilliseconds` when nothing has arrived.
4. Reads the clock.
5. Takes what arrived: the events the socket goroutines put on the arrival
   channel up to its length at that moment, a listen or UDP dial result, and
   the dropped counts.
6. Gives each accepted connection the lowest free handle, in the order of
   the listener handles and then of acceptance.
7. Advances the timer wheel to the clock and adds a timeout entry for each
   timeout that expired.
8. Sorts the batch by socket handle, stable, so the entries of one socket
   keep their arrival order (decision D49).

Every step works on the entries and the sockets that have one, never on the
open sockets as a whole, so the work of an update grows with the ready count
and not with the number of sockets. `TestNetUpdateWorkGrowsWithReady` in
`t1-netbatch` counts it, and `BenchmarkNetUpdate` times it. Measured on an
Intel i7-3770S, the time of `net.Update` alone, in ns:

| Sockets open | 1 ready | 16 ready | 64 ready |
| --- | --- | --- | --- |
| 16 | 2800 | 9100 | |
| 256 | 2900 | 9600 | 45500 |
| 1000 | 2800 | 10600 | 45100 |

## The runtime

The runtime is `t1-netbatch`, which needs no compiler. Each socket has one
reader goroutine, which Go's netpoller parks until data comes (epoll, kqueue
or IOCP), and one writer goroutine. A reader reads into the socket's receive
ring of 16 KiB and puts one event on the arrival channel, of 4096 events.
The update drains the channel into the batch. Handles, the ring, the send
buffers and the batch are fixed in size and made once, when the library
registers or when a socket opens.

| Limit | Size |
| --- | --- |
| handles | 1023, `NetSocketCountMax` |
| receive ring per socket | 16 KiB, `NetRingSize` |
| written per socket between two updates | 16 KiB, `NetSendSize` |
| arrival channel | 4096 events, `NetArrivalCountMax` |
| UDP peers | 1023, `NetPeerCountMax` |

When a ring or the channel is full, a TCP reader stops reading its socket
until an update frees space, and TCP flow control slows the peer. A UDP
datagram that does not fit is dropped and counted in a dropped entry
(decision D50). Only the program takes and frees handles: an open takes the
lowest free one, an update numbers the accepted connections, and a close
frees it at once. An event of a socket the program closed is dropped, and a
handle opened again never meets one of the old socket.

`net.Write` queues nothing before the program has seen the socket connected,
accepted or bound, so how much a program may write depends only on what it
has seen. A UDP peer is numbered the first time the program meets its
address, in a received datagram or `net.UDPPeerResolve`, so peers too depend
only on the batches.

## The replay record

Every update is the only point where the network reaches a program, so the
batches are the whole network input of a run. A live run writes them with
`--net-record`, and a replay reads them with `--net-replay`, opens no socket
and prints what the live run printed, on the old loop and on vm (decisions
D48 and D53).

```
cx --enable=net --net-record=tests/test-net-tcp.netreplay tests/test-net-tcp.cx
cx --enable=net --net-replay=tests/test-net-tcp.netreplay tests/test-net-tcp.cx
```

A program built with `cx build` takes no cx flags, so it replays the record
the environment variable `CX_NET_REPLAY` names.

The record is a text sections file. The `updates` section has one line per
update, `u number clock`, with the clock reading the update gave (decision
D51), and after it one line per entry of its batch,
`e handle trigger detail peer payload`. The peer is the UDP address of a data
entry or `-`, and the payload the quoted bytes of a data entry or `-`.

```
[format] version
1

[updates] kind fields
u 1 1
e 1 4 38221 - -
u 2 2
e 1 2 3 - -
e 2 3 0 - -
u 3 2
e 3 1 2 - "hi"
```

A replay checks nothing the program sends, which is not output. A program
that asks for an update past the end of the record, or whose record names a
socket it did not open, ends with an assert, since it is not the program
that was recorded. The execution records replay the record beside a program,
see [ExecutionRecords.md](ExecutionRecords.md).

## Timeouts

`net.TimeoutSet` keeps one timeout per handle in a hashed timer wheel of 256
slots of one millisecond (Varghese and Lauck, "Hashed and hierarchical timing
wheels", SOSP 1987). Starting and stopping cost O(1), and an update costs the
slots it passes, at most one turn, and the timeouts in them. A timeout
expires at the first update whose clock reading is at or past it, as a
timeout entry. The clock is the monotonic time since the run started, read
once per update (decision D51).

## Allocations

`net.Update`, `net.ReadyCount`, `net.ReadyNext`, `net.Clock`, `net.Read`,
`net.Write`, `net.UDPReadFrom` and `net.UDPWriteTo` make no Go allocation.
`TestNetUpdateAllocs` in `t1-netbatch` requires 0 for an update with data on
four sockets, the reads and the writes, and `TestRunNetAllocs` and
`TestRunVmNetAllocs` in `t11-cx` require 0 for a CX program that calls them
on the old loop and on vm. Opening a socket allocates its buffers, its
goroutines and Go's `net.Conn`, VMALLOC-018, and so does `net.UDPPeerResolve`
parsing its address. Those happen in the open, not in an update.

## The vm

Every function has a vm form. A `str` input is read from the data segment,
and a buffer is lowered as the address and the byte size of the array, which
the vm form reads or fills in place, as the old loop does. A program that
builds an address with `sprintf` does not lower, which is why the address
functions take the host and the port apart.

## js/wasm

The library compiles for js/wasm. Go's `net` package there is an in process
fake that exists for tests (`net_fake.go`), so a program reaches only
sockets it opened itself in the same page. A browser has no TCP or UDP. A
real peer needs a relay the page reaches over a WebSocket, which is not
built.
