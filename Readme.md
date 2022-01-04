# Some custom Scheme Lisp dialect interpreter

This is an implementation of a custom dialect of Lisp in golang. GNU Guile and
Racket Scheme implementations common behavior has been used as a reference.

The work is still in progress.

Sort of roadmap:
- [x] Lexing (tokenization)
- [x] Parsing (building internal AST)
- [ ] Evaluation (interpreting)
  - [x] Quotation
  - [x] `car`, `cdr`
  - [x] Arithmetic operations (currently only `+` implemented)
  - [ ] `define` and lexical scoping
    - [ ] Constants
    - [ ] Procedures
  - [ ] `lambda`

Maybe it is worth to implement:
- Quasiquote and unquote
- `define-syntax`

What won't be implemented (almost certainly)
- Floating point numbers
- UTF-8 and any encodings beyond ASCII, even for strings and comments

Sometimes I do live streaming of the development process on ["\[RU\] \*nixtalks" Discord
server](https://discord.gg/dDPfD5SFDR).
