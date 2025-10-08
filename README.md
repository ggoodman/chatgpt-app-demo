## Create and configure an Auth0 tenant

1. Go to https://manage.auth0.com and sign up or log in. In the tenant selector, choose to create a new tenant.

   ![Tenant creation screen](https://github.com/user-attachments/assets/14453ba7-1917-44c6-b2aa-572a9dbf93b5 "Tenant creation screen")

2. In the `Applications` section on the sidebar pick `APIs`. We want to define our MCP Server as a backend API. On the APIs page, click the "Create API" button.

   When creating the API, there are a few critical nuances:

   - The "Identifier" must match the full URL at which the MCP server will live. In this demo, it is hosted on the `/mcp` path so I include that in the API's identifier.
   - I have selected the `RFC 9068` JWT profile. [RFC 9068](https://datatracker.ietf.org/doc/rfc9068/) is a spec by my late colleague Vittorio Bertocci that defines a JWT-based format for Access Tokens. This will allow our MCP Server code to validate Access Tokens using using public keys it can discover via the public JWKS endpoint and then extract the `sub` claim that represents the authenticated user's id.

   ![API creation screen](https://github.com/user-attachments/assets/2e06f7d0-1986-4ada-bdc0-232f239ef970 "Creating an API for our MCP Server")

3. Next, we can create a "Social Connection", use the built-in "Username-Password-Authentication" database connection or get fancy with other options offered by Auth0. For example, if you wanted to authenticate users using their GitHub identities, you might setup the [GitHub Integration](https://marketplace.auth0.com/integrations/github-social-connection).

   The key outcome here is that we have a Connection we intend to use to for logging our users in. The Connection defines where the main login 'factor' for the user comes from. In the case of the "Username-Password-Authentication" Connection, users come from a database Auth0 manages and such users identify themselves using a combination of username and password. For a GitHub connection, Auth0 is configured to trust GitHub as a federated OpenID identity provider.

   We need the identifier for this connection. It starts with `con_` and can be found on that Connection's page in the header. Now we need to configure it as a "domain connection". A "domain connection" simply means that new, dynamic clients will be automatically configured to accept identities from that connection. Normally, you need to explicitly decide which Connections will be allowed to provide identities to which Apps. In the MCP world, a Server creator doesn't know all of the different AI apps that might want to connect to it; these apps will dynamically register themselves as needed.

   > ![NOTE]
   > As a pre-requisite to this, please install the `auth0` CLI and configure it for your tenant via the `auth0 login` flow.

   ```sh
   auth0 api patch "connections/con_<REDACTED>" --tenant <your_tenant>.auth0.com --data '{ "is_domain_connection": true }'
   ```

4. Navigate to the `Settings` section via the sidebar. Scroll down to "API Authorization Settings" and enter your MCP Server's URL (the identical value to the API `identifier` from step 2) in the field, "Default Audience" and then click "Save".

   ![Default audience](https://github.com/user-attachments/assets/ba1e7c89-bdc9-43fb-a2d0-be477e2732ca "Configure default audience")

   This configuration tells Auth0 to assume that tokens should be issued for use in our MCP server if no audience was requested in an authorization flow.

5. This time in the "Advanced" tab of "Tenant Settings", we will enable dynamic client registration. The 2025-06-18 version of the MCP Spec [requires that clients dynamically register](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization#dynamic-client-registration) themselves using this "DCR" mechanism.

   ![Enable DCR](https://github.com/user-attachments/assets/764dd1de-12e4-4f8a-bdfb-9f39d25ba3e9 "Enable OIDC Dynamic Client Registration")

All done. Now you have the full power of Auth0 and its myriad connectors and capabilities at your MCP Server's disposal.

## Project set-up

Create a new project and initialize it as a go module.

```sh
# Adjust according to your use-case.
go mod init github.com/ggoodman/chatgpt-app-demo
```

### Install mcp-server-go

```sh
go get -u github.com/ggoodman/mcp-server-go
```

### Define your MCP server capabilities

```go
tools := mcpservice.NewToolsContainer(
  // Add your tools here
)

// Use string concatenation to safely include fenced code block without confusing the Go parser.
detailedInstructions := `<TODO>`

return mcpservice.NewServer(
  mcpservice.WithServerInfo(
    mcpservice.StaticServerInfo("Example ChatGPT App", "0.0.1", mcpservice.WithServerInfoTitle("Example ChatGPT App")),
  ),
  mcpservice.WithProtocolVersion(mcpservice.StaticProtocolVersion("2025-06-18")),
  mcpservice.WithInstructions(mcpservice.StaticInstructions(detailedInstructions)),
  mcpservice.WithToolsCapability(tools),
)
```

### Define your MCP tools

TODO

### Mount your MCP Service as an HTTP Handler

Here, we're creating a redis "Session Host". A session host is responsible for wiring messages between instances, persisting session events and managing session storage and lifecycle. The `mcp-server-go` SDK also has a "Memory Host" but that is only suitable when there will be exactly one instance per session (such as a CLI or singleton web server).

This is also where we're instantiating an auth provider for our MCP Server. You'll notice that it's as simple as supplying the full URL of our MCP Server (`https://chatgptapp.goodman.dev/mcp`) and that of our Authorization Server (`https://chatgpt-app-demo.us.auth0.com`). The SDK will automatically configure the tricky details of MCP Authorization for you.

Finally, we're combining our MCP Service, Session Host and Auth Provider together and mounting them into a `StreamingHTTPHandler`.

```go
redisHost, err := redishost.New(redisUrl, redishost.WithKeyPrefix(redisKeyPrefix), redishost.WithStreamMaxLen(20))
if err != nil {
  return nil, fmt.Errorf("error instantiating redis host: %w", err)
}

auth, err := auth.NewFromDiscovery(ctx, authIssuerUrl, serverUrl)
if err != nil {
  return nil, fmt.Errorf("error configuring auth: %w", err)
}

srv := NewMCPServerCapabilities()

return streaminghttp.New(ctx, serverUrl, redisHost, srv, auth,
  streaminghttp.WithServerName("Example ChatGPT App"),
  streaminghttp.WithLogger(log),
  streaminghttp.WithVerboseRequestLogging(true),
)
```
