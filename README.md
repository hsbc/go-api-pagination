# go-api-pagination

## Introduction

The `pagination` package provides a simple and efficient way to handle pagination with the GitHub API. It offers interfaces and functions to list, process, and handle rate limits for any type of items you wish to paginate through.

## Getting Started

### Installation

To get started with the `pagination` package, you can install it using the `go get` command:

```bash
go get github.com/hsbc/go-api-pagination
```

### Importing the Package

After installation, you can import the package in your Go code:

```go
import "github.com/hsbc/go-api-pagination/pagination"
```

## Usage

### Interfaces

1. **ListFunc**: This interface returns a list of items for the given type. It requires the implementation of the `List` method which fetches the items.

2. **ProcessFunc**: This interface processes an item for the given type. It requires the implementation of the `Process` method which processes each item.

3. **RateLimitFunc**: This interface handles rate limiting. It requires the implementation of the `RateLimit` method which decides whether to continue pagination based on rate limits.

### Paginator Function

The main function provided by this package is `Paginator`. It takes in the following parameters:

- `ctx`: The context for the API calls.
- `listFunc`: An instance of a type that implements the `ListFunc` interface.
- `processFunc`: An instance of a type that implements the `ProcessFunc` interface.
- `rateLimitFunc`: An instance of a type that implements the `RateLimitFunc` interface.
- `Opts`: An instance of `PaginatorOpts` which contains list options like `PerPage` and `Page`.

The function paginates through the items, processes them, handles rate limits, and returns all the items.

### Example Implementations

Here are example implementations for the interfaces:

```go
type listFunc struct {
	client *github.Client
}

func (l *listFunc) List(ctx context.Context, opt *github.ListOptions) ([]*github.Repository, *github.Response, error) {
	t, r, err := l.client.Apps.ListRepos(ctx, opt)
	return t.Repositories, r, err
}

type processFunc struct {
	client *github.Client
}

func (p *processFunc) Process(ctx context.Context, item *github.Repository) error {
	fmt.Println(item.GetName())
	return nil
}

type rateLimitFunc struct {
}

func (r *rateLimitFunc) RateLimit(ctx context.Context, resp *github.Response) (bool, error) {
	if resp.Rate.Remaining <= 1 {
		time.Sleep(time.Until(resp.Rate.Reset.Time))
	}
	return true, nil
}
```

### Using the Paginator Function with Example Implementations

```go
// Initialize the GitHub client
githubClient := github.NewClient(nil)

// Define your list, process, and rate limit functions
listFuncInstance := &listFunc{client: githubClient}
processFuncInstance := &processFunc{client: githubClient}
rateLimitFuncInstance := &rateLimitFunc{}

// Call the Paginator function
items, err := pagination.Paginator(ctx, listFuncInstance, processFuncInstance, rateLimitFuncInstance, &pagination.PaginatorOpts{
    ListOptions: &github.ListOptions{PerPage: 50, Page: 1},
})
if err != nil {
    log.Fatalf("Error paginating: %v", err)
}

// Process the items
for _, item := range items {
    // Your logic here
}
```

## Conclusion

The `pagination` package simplifies the process of paginating through items from the GitHub API. By providing clear interfaces, example implementations, and a main Paginator function, it abstracts away the complexities of pagination, rate limiting, and item processing.
