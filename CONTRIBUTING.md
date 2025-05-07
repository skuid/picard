# Contributing to skuid/picard

This is an open source project started for internal use at Skuid, aimed at providing a full-featured ORM that enforces multi-tenancy by default.

## Filing issues

When filing an issue, make sure to answer these five questions:

1. What version of Go are you using (`go version`)?
2. What operating system and processor architecture are you using?
3. What did you do?
4. What did you expect to see?
5. What did you see instead?

## Contributing code

Before submitting changes, please follow these guidelines:

1. Check the open issues and pull requests for existing discussions.
2. Open an issue to discuss a new feature.
3. If discussion ends up recommending the feature, fork this repository.
4. Implement the new feature in your fork.
5. If you add new dependencies, please update `go.mod` and run `go get`.
6. Write tests.
7. Create an upstream pull request against the skuid/picard repository, and make sure Github Actions CI checks pass.
