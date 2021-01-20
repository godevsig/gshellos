fmt := import("fmt")
sh := import("shell")

fmt.println(ex("hello world").split().join(", "))
fmt.println(ex("ni hao gshell").split("g")[0].split())

cdr := sh.run(`echo hello world`)
show(type_name(cdr.output))
fmt.println(cdr.output)
show(cdr.output)

estr := ex("1,2,3,4.5.6.7.8")
show(type_name(estr))
show(estr)
show(estr[4])

earray := estr.split(",", ".")
show(type_name(earray))
show(earray)
show(earray[5])

et := ex([1,3,5,7])
show(et)
et = ex(et)
show(et)
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
