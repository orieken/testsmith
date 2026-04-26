; Top-level function definitions (async and sync)
(module
  (function_definition
    name: (identifier) @func.name
    parameters: (parameters) @func.params) @func)

(module
  (decorated_definition
    (function_definition
      name: (identifier) @func.name
      parameters: (parameters) @func.params) @func))

(module
  (async_function_def
    name: (identifier) @func.name
    parameters: (parameters) @func.params) @func)

; Top-level class definitions
(module
  (class_definition
    name: (identifier) @class.name
    body: (block) @class.body) @class)

(module
  (decorated_definition
    (class_definition
      name: (identifier) @class.name
      body: (block) @class.body) @class))

; Methods inside classes (sync and async)
(class_definition
  body: (block
    (function_definition
      name: (identifier) @method.name
      parameters: (parameters) @method.params) @method))

(class_definition
  body: (block
    (async_function_def
      name: (identifier) @method.name
      parameters: (parameters) @method.params) @method))
