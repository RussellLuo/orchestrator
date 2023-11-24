# Orchestrator

A Go library for service orchestration, inspired by [Conductor][1].


## Core Concepts

### Task

Tasks are the fundamental building blocks of Orchestrator. They are similar to primitive types or statements in a programming language.

Typically, a [task](task.schema.json) accepts an input and returns an output. Each parameter in the input can be a literal value or an [expression](#expression).

Built-in tasks:

- [Decision](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Decision)
- [Terminate](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Terminate)
- [Raise](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Raise)
- [Loop](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Loop)
- [Iterate](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Iterate)
- [HTTP](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#HTTP)
- [Serial](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Serial)
- [Parallel](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Parallel)
- [Call](https://pkg.go.dev/github.com/RussellLuo/orchestrator/builtin#Call)


### Flow

A flow is used to define a piece of logic, which usually consists of one or more tasks. Flows are similar to functions or routines in a programming language.

In Orchestrator, a flow is essentially a composite task (i.e. a `Serial` task). Therefore, just like a task, a flow accepts an input and returns an output. Furthermore, a flow can be embedded into another flow by leveraging a `Call` task, thus serving as a sub-flow.

### Expression

Expressions are used to extract values out of the flow input and other tasks in the flow.

For example, a flow is supplied with an input by the client/caller when a new execution is triggered. The flow input is available via an expression of the form `${input.<var>}` (see [dialects](#dialects)). Likewise, the output of a previously executed task can also be extracted using an expression (e.g. `${<task>.<output_var>}`) for use in the input of a subsequent task.


#### Dialects

Currently supported expression dialects:

- [Starlark][3]

    ```
    ${input.value}  // Value from the input.
    ${tool.status == 200}  // Whether the status code (from an HTTP task `tool`) is 200
    ${len(tool.body.entities)}  // Length of the response entities (from an HTTP task `tool`)
    ${[s*2 for s in input.scores]}  // List comprehension
    ${{k: v.upper() for k, v in input.properties.items()}}  // Dictionary comprehension
    ```

- [Expr][4]

    ```
    #{input.value}  // Value from the input.
    #{tool.status == 200}  // Whether the status code (from an HTTP task `tool`) is 200
    #{len(tool.body.entities)}  // Length of the response entities (from an HTTP task `tool`)
    //#{[s*2 for s in input.scores]}  // UNSUPPORTED
    //#{{k: v.upper() for k, v in input.properties.items()}}  // UNSUPPORTED
    ```

- JSONPath ([spec][5] and [implementation][6])

    ```
    @{input.value}  // Value from the input.
    //@{tool.status == 200}  // UNSUPPORTED
    //@{len(tool.body.entities)}  // UNSUPPORTED
    //@{[s*2 for s in input.scores]}  // UNSUPPORTED
    //@{{k: v.upper() for k, v in input.properties.items()}}  // UNSUPPORTED
    ```


## Documentation

Checkout the [Godoc][2].


## License

[MIT](LICENSE)


[1]: https://github.com/Netflix/conductor
[2]: https://pkg.go.dev/github.com/RussellLuo/orchestrator
[3]: https://github.com/google/starlark-go/blob/master/doc/spec.md#expressions
[4]: https://expr.medv.io/
[5]: https://goessner.net/articles/JsonPath/
[6]: https://github.com/PaesslerAG/jsonpath
