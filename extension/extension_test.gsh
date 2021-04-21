fmt := import("fmt")
sh := import("shell")

fmt.println(ex("hello world").split().join(", "))
fmt.println(ex("ni hao gshell").split("g")[0].split())

cdr := sh.run(`echo hello world`)
help(type_name(cdr.output))
fmt.println(cdr.output)
help(cdr.output)

estr := ex("1,2,3,4.5.6.7.8")
help(type_name(estr))
help(estr)
help(estr[4])

earray := estr.split(",", ".")
help(type_name(earray))
help(earray)
help(earray[5])

et := ex([1,3,5,7])
help(et)
et = ex(et)
help(et)
et = ex(ex("hhh"))


//output:
//hello, world
//["ni", "hao"]
//"efunction: output"
//<efunction>
//<efunction>:
//Signature
//    output() -> string
//Usage
//    Get the output of the commander obj, will bock until commander exits.
//Example
//    fmt.println(cdr.output())
//
//"estring"
//1,2,3,4.5.6.7.8
//3
//"earray"
//["1", "2", "3", "4", "5", "6", "7", "8"]
//6
//[1, 3, 5, 7]
//[1, 3, 5, 7]
