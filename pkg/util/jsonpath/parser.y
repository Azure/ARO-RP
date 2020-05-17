%{
package jsonpath
%}

%union {
    r  rule
    rs rules
    s  string
}

%token<s> STRING

%type<rs> rules
%type<r>  rule

%%

jsonpath: '$' rules                             { yylex.(*lexer).out = $2 }

rules:                                          { $$ = rules{} }
     | rules rule                               { $$ = append($1, $2) }

rule: '.' STRING                                { s := $2; $$ = &subscript{name: &s} }
    | '.' '*'                                   { $$ = &subscript{} }
    | '[' STRING ']'                            { s := $2; $$ = &subscript{name: &s} }
    | '[' '*' ']'                               { $$ = &subscript{} }
    | '[' '?' '(' '@' rules '=' STRING ')' ']'  { $$ = &filter{r: $5, s: $7} }

%%
