# elastic-agent-inputs

The code in this repository was an experiment that was never released, with no planned future development.

## Project structure

### `inputs`
`input` is the folder containing all inputs, each input must be on its
own package.

Each input must define the following functions and types:
 - `Config` (type): it's config struct
 - `Command` (function): a function that returns a ``*cobra.Command`,
   it must accept a logger and its configuration struct.
 - `DefaultConfig` (function): Returns the config struct populated
   with the default values.

### `pkg`
All common packages/functionalities go into this package
