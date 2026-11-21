this is first draft

The CX grammar is defined in two yacc files, [t5-cxpartialparsing/cxpartialparsing.y](../t5-cxpartialparsing/cxpartialparsing.y) for the first pass and [t8-cxparsingcompletor/parsingcompletor.y](../t8-cxparsingcompletor/parsingcompletor.y) for the second.

cx grammer is used to generate cx parser which is used to generate program ast.

`make build-parser` passes them to goyacc to regenerate the parsers:

```
go tool goyacc -o t5-cxpartialparsing/cxpartialparsing.go -v /dev/null t5-cxpartialparsing/cxpartialparsing.y
go tool goyacc -o t8-cxparsingcompletor/parsingcompletor.go -v /dev/null t8-cxparsingcompletor/parsingcompletor.y
```


cx grammer is defined 

declarations of variable

rules of action 

functions



cx grammer consists of two things 

BNF grammar  - > action

Whenever a particular BNF grammar rule is recognized, the action describes what to do is perform on current token. 

For example

function_declaration:
                function_header function_parameters compound_statement
                {
			actions.FunctionDeclaration($1, $2, nil, $3)



if we found grammer function_declaration at the time of parsing then its corresponding action is perform 
actions.FunctionDeclaration()

All CX actions are defined in [t4-actions](../t4-actions).


at the end of ParseSourceCode() we get complete CXProgram as object.


 