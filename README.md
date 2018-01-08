# backoff

Package backoff implements a simple backoff algorithm, executing
a function repeatedly until it returns a non-error result or
the maximum allowed number of retries has been reached.

The package implement two different strategies: 
- `backoff.Linear`: Attempts are made linearly, depending on maxRetries
- `backoff.Exponential`: Attempts are made Exponentially, increasing the time between retries

