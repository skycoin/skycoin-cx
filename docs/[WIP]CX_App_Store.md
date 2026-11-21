# CX App Store
Planned repository: skycoin/cx-app-store. It was never created. Work on CX itself is in [skycoin/skycoin-cx](https://github.com/skycoin/skycoin-cx).

CX App Store will be a downloadable app for windows and mac. Once it is downloaded and installed, when ran, the app will start its own local web app server and automatically open up a web browser that redirects to that web app. It will show lists of CX programs/apps/games that the user can run. For example, the user clicks "CXPacman", it will then automatically open up an OpenGL window that runs the CXPacman game.

- Task Bar Launcher (Rudi has working)
- Small web-app
- Server/Key-value/DHT for packages (Redis superseded by bolt or a minimalist key-value store, the smaller the better)
- All static files run locally. Runs by JSON files in a directory. All offline?
    – based upon apple: jet a list of apps, with a page for each app; and title+text 

Interface:
- Game runs with OpenGl window (most existing apps are games)
- Eventually: iPython notebook + terminal interface for interacting?

