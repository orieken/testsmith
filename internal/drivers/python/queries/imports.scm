; Captures regular imports: import os  /  import os as operating_system
(import_statement
  name: (dotted_name) @import.module)

(import_statement
  name: (aliased_import
    name: (dotted_name) @import.module
    alias: (identifier) @import.alias))

; Captures from-imports: from os import path  /  from os import path as p
(import_from_statement
  module_name: (dotted_name) @import.module
  name: (import_from_names
    (identifier) @import.name))

(import_from_statement
  module_name: (dotted_name) @import.module
  name: (import_from_names
    (aliased_import
      name: (identifier) @import.name
      alias: (identifier) @import.alias)))

; Relative imports: from . import foo  /  from .utils import helper
(import_from_statement
  module_name: (relative_import) @import.relative
  name: (import_from_names
    (identifier) @import.name))

; Wildcard: from os import *
(import_from_statement
  module_name: (dotted_name) @import.module
  name: (wildcard_import) @import.wildcard)
