# orchestrator

A Go library for service orchestration, inspired by [Conductor][1].


## Core Concepts

### Task

Tasks are the fundamental building blocks of Orchestrator. They are similar to primitive types or statements in a programming language.

Typically, a task accepts an input and returns an output. Each parameter in the input can be a literal value or an [expression](#expression).

Built-in tasks:

- [Decision](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Decision)
- [Terminate](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Terminate)
- [Loop](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Loop)
- [Iterate](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Iterate)
- [HTTP](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#HTTP)
- [Serial](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Serial)
- [Parallel](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Parallel)
- [Call](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Call)


### Flow

A flow is used to define a piece of logic, which usually consists of one or more tasks. Flows are similar to functions or routines in a programming language.

In Orchestrator, a flow is essentially a composite task (i.e. a `Serial` task). Therefore, just like a task, a flow accepts an input and return an output. Furthermore, a flow can be embedded into another flow by leveraging a `Call` task, thus serving as a sub-flow.

### Expression

Expressions are used to extract values out of the flow input and other tasks in the flow.

For example, a flow is supplied with an input by the client/caller when a new execution is triggered. The flow input is available via an expression of the form `${input.<var>}` (see [dialects](#dialects)). Likewise, the output of a previously executed task can also be extracted using an expression (e.g. `${<task>.<output_var>}`) for use in the input of a subsequent task.


#### Dialects

Currently supported expression dialects:

- JSONPath ([spec][3] and [implementation][4])

    ```
    ${tool.status}  // HTTP status code (e.g. 200) - from an HTTP task named `tool`.
    ```

- [Expr][5]

    ```
    #{len(tool.body.entities)}  // Length of returned entities (e.g. 10) - from an HTTP task named `tool`.
    ```


## Documentation

Checkout the [Godoc][2].


## License

[MIT](LICENSE)


[1]: https://github.com/Netflix/conductor
[2]: https://pkg.go.dev/github.com/RussellLuo/orchestrator
[3]: https://goessner.net/articles/JsonPath/
[4]: https://github.com/PaesslerAG/jsonpath
[5]: https://expr.medv.io/
