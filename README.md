# Go Logger
An extremely fast logger and secure binary communication protocol for log entries, based on TCP and TLS 1.3. Consists of a client and a server.

## Concept
1. The client and server has a private key (ED25519) each.
2. A certificate authority (CA) issues a TLS certificate for both the client and the server, based on their public keys.
3. The root CA certificate is provided to both the client and the server.
4. The client connects to the server over TCP, secured with TLS 1.3.
5. Both parties (client and server) verifies that the other party's certificate actually was signed by the CA.
6. The client starts streaming log entries over the connection, and keeps the connection open for as long as needed.

## Usage
- [Client](./docs/client.md)
- Server (docs coming soon)