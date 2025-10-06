# Yuna

A library for building HTTP applications and microservices in GO.

<p align="center">
  <img src=".github/images/IMG_7794.jpg" alt="Yuna" width="500" />
</p>

Naming things is hard, way harder than it should be. Or perhaps I just suck at naming. Anyway, Yuna is the name of our family cat, and the name of a well-known character from a Final Fantasy game. It didn't seem like a common name or conflicting with other projects, so we went with it.

## Philosophy and Design Goals

* Provide a clean, simple, and intuitive API
* Maintain compatibility with the standard library http package and existing middlewares
* Provide out-of-the-box metrics, tracing, logging
* Implement RFC 7807 / RFC 9457 for standard error responses and validation errors
* Make it easy to bring your own implementations of core concepts in Yuna, such as the `Responder` interface

## Core Concepts

### Handler & Responder

The `Handler` interface is the core concept of Yuna. It defines a single method that takes a `Request` and returns a `Responder`. Unlike the http.Handler in the standard library, the handlers in Yuna do not directly write the response to the client. Instead, they return a `Responder` which Yuna will invoke to write the response. This provides a thin abstraction allowing the handlers to focus on what should be returned to the client, rather than specifics.

Yuna provides two implementations of the `Responder` interface: `ResponseBuilder` and `ProblemDetails`. ResponseBuilder is a type for building a response to be sent to the client. ProblemDetails is a type for building an error response to be sent to the client following RFC 7807 / RFC 9457. Both of these types have several helper functions for building common responses. ResponseBuilder supports reponding with text/html and application/json content types. ProblemDetails supports reponding with application/problem+json content type.

Examples:

```go
// todo: add examples of each here
```

Since Responder is an interface, you can implement your own custom Responder types to suit your needs. This could be useful if you want to support responding with a different content type such as application/xml, application/msgpack, or application/protobuf. This could also be useful if you want to support content-negotiation and respond with different content types based on the client's Accept header. Additionally, a custom or standard logic such as setting default headers, etc. could be implemented in a Responder. If you don't want to follow Problem Details RFC 7807 / RFC 9457, you can implement your own Responder for standardized error responses.

### Request ###

The `Request` type is a thin wrapper around the standard http.Request type. It aims to provide a simple API and reduce boilerplate code for getting query parameters and path parameters from the request. The QueryParam and PathParam methods return a ParamValue type, which provides convenient methods for getting the value of the parameter as a string, an int, a float64, or a bool. This helps reduce boilerplate code for parsing and converting query parameters and path parameters to their appropriate types. Additionally, `Request` provides a Bind method for binding query/form parameters to a struct.

### Yuna ###

Yuna is the main type in the library. It handles routing, middleware, metrics, tracing, logging, and how we start the server. `Yuna` differs from many other libraries and frameworks as it uses two http.Server(s) under the hood. There is the main application server (default port is 8080) and an operational server (default port is 8082). The operational server is used for health checks, metrics, pprof, along with info and uptime endpoints. The main application server is used for serving the handlers registered with Yuna. This helps keep clear separation of concerns and allows for more flexibility in how the application is deployed. 

Yuna will automatically configure OpenTelemetry for tracing and metrics of all requests. It also creates a scoped logger for each request that can be retrieved using the `log.LoggerFromCtx` function. As for middleware, under the hood Yuna uses the [chi](https://github.com/go-chi/chi) router. It wraps a yuna.Handler in an http.Handler and registers it with the chi router. This means that Yuna works with all middlewares that adhere to standard http.Handler interface. Milddlewares can be registered/configured with `Use` or `With`, or on individual routes when they are registered. 