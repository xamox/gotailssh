# Go SSH Proxy via Tailscale

This project implements an SSH proxy server that runs over a Tailnet using Tailscale's [`tsnet`](https://pkg.go.dev/tailscale.com/tsnet) package. It creates an ephemeral SSH server which accepts SSH connections and spawns a shell session inside a pseudo-terminal.

## Overview

- **SSH Server**: Listens for SSH connections on port 2222 within the Tailnet.
- **Authentication**: Uses environment variables for authentication:
  - `TS_AUTHKEY`: Tailnet authentication key.
  - `SSH_AUTHORIZED_KEY`: SSH public key that is allowed to connect.
- **State Management**: Uses the `state/` directory to store ephemeral state for the Tailnet server.
- **Pseudo-Terminal**: Supports starting a shell session with a pseudo-terminal using the [`github.com/creack/pty`](https://pkg.go.dev/github.com/creack/pty) package.

## Requirements

- Go 1.23 or later.
- A valid Tailscale auth key (`TS_AUTHKEY`).
- An authorized SSH public key (`SSH_AUTHORIZED_KEY`).

## Getting Started

1. **Set Environment Variables**  
   Ensure you have the necessary environment variables before running the application:
   - `TS_AUTHKEY`: Your Tailnet authentication key.
   - `SSH_AUTHORIZED_KEY`: The public key for SSH authentication (in authorized key format).

2. **Run the Application**  
   Use the following command to build and run the project:

   ```sh
   go run main.go
   ```

   The server will start and listen on port `2222` for incoming SSH connections within your Tailnet.

3. **SSH Connection**  
   Use your SSH client to connect to the Tailnet address on port 2222. For example:

   ```sh
   ssh -p 2222 user@<tailnet-ip-address>
   ```

## Additional Details

- **State Directory**:  
  The `state/` directory holds the persistent state needed by the Tailnet server. The `.gitignore` file is configured to ignore this directory.

- **Ephemeral Server**:  
  The server is designed to be ephemeral. It cleans up its state automatically when it goes offline.

- **Logging**:  
  The application logs incoming connections and errors, which can be viewed in the terminal or application logs.

## License

This project is provided as-is. For more information about Tailscale’s licensing, please refer to their [official documentation](https://tailscale.com/).

Happy coding!