## og-preview

This sample Go application fetches the `og:` tags from a given
website. It currently runs as an HTTP server with a single endpoint,
`/get_tags`. 

It uses Redis for client-side caching, and it expects a server running
on localhost on the default (6379) port.
