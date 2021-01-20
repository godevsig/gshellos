fmt := import("fmt")
sh := import("shell")

fmt.print(sh.run(`echo hello world | grep world`).output())

cdr := sh.run(`rm nofile`)
cdr.wait(5)
cdr.wait()
fmt.print(cdr.output())
fmt.println(cdr.errcode())

cdr = sh.run(`ping localhost`)
cdr.wait(1)
cdr.kill()
fmt.println(cdr.errcode())

//output:
//hello world
//rm: cannot remove 'nofile': No such file or directory
//1
//-1

