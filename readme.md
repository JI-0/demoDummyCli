# Demo Dummy WebRTC Client server
This is a Dummy WebRTC Client server, that is used to visualize the Dummy WebRTC client system explained [here](https://ji-0.github.io/posts/webrtc/).
This project can be either used and run from the source code, or as a built server application by running `go build .`.

Currently only a release for macOS with Apple Silicone is available, as other OSs are untested.

## Functionality
This server opens a socket on port 3002 and listens for connections. Once the website connects to the localhost when the "Start dummy cli test" putton is pressed, the server sends notifications of new packets arriving with the dummy client number and timestamp, and when a new peer is added or a second passes.

## Known issues
There are currently two known issues with this server:
* The library reports a broke pipe, which is part of the old code that has to be removed, which writes data to a .csv file.
* Some browsers do not allow a connection to localhost from the website, [https://ji-0.github.io/posts/webrtc/](https://ji-0.github.io/posts/webrtc/), so it might require the user to allow this connection or to run the website locally by cloning the repo: [https://github.com/JI-0/JI-0.github.io](https://github.com/JI-0/JI-0.github.io), and running it by first installing dependencies `npm install` and then running it `npm run dev`.