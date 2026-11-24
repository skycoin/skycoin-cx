
## Glossary
* [AST](https://en.wikipedia.org/wiki/Abstract_syntax_tree) - An abstract syntax tree (AST) is a way of representing the syntax of a programming language as a hierarchical tree-like structure. The AST contains a final parsed output that can be executed.
* [Lexer](https://en.wikipedia.org/wiki/Lexical_analysis) - takes a text [or sequence of characters] as an input and breaks it up into a list of tokens.
* [CXParser Spec](cxparser_spec.md) - CX Parser Specification defines the cx parser.
---

# CX Compiler Stages

Parsing starts in [t11-cx/helpers.go parseProgram()](../t11-cx/helpers.go). This creates an empty AST and reads the CX source code into strings and calls compileSourceStrings(), which calls [t9-cxparsing/cxparsing.go ParseSourceStrings()](../t9-cxparsing/cxparsing.go). The strings are passed to [t9-cxparsing/utils.go Preliminarystage()](../t9-cxparsing/utils.go). 

`Preliminarystage`

In [t9-cxparsing/utils.go Preliminarystage()](../t9-cxparsing/utils.go), it will identify all the packages, structs, global variables, and package imports and add these to our AST. The comments are disregarded. After this, PassOne() will be called.

`PassOne`

PassOne or the first parsing stage, using [t5-cxpartialparsing/cxpartialparsing.go Parse()](../t5-cxpartialparsing/cxpartialparsing.go), compiles function primitives and types, along with the structure of their parameters and global variables. This stage also finalizes compilation of structs and ensures packages and their imports are correct. This uses `goyacc` for parsing and creates a chain of tokens for the parser using an in-house lexer known as `Lexer`. 

`PassTwo`

The second parsing stage, PassTwo or the second parsing stage, using [t8-cxparsingcompletor/parsingcompletor.go Parse()](../t8-cxparsingcompletor/parsingcompletor.go), fully compiles functions and all expressions. 

Finally, the `main` function and invisible `*init` functions are created, the latter of which acts as an initializer for global variables. After this, the CXProgram AST is complete and is ready to run.
