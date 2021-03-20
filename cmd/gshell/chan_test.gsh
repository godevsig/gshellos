fmt := import("fmt")

c1 := makechan(3)
c2 := makechan(2)

c1.send("hello")
c1.send("godevsig")

c2.send("nihao")
c2.send(123)

fmt.println(c1.recv())
fmt.println(c1.recv())
fmt.println(c2.recv())
fmt.println(c2.recv())

c1.send(5.56)
fmt.println(c1.recv())

c1.send("before close1")
c1.send("before close2")
c1.close()
show(c1.recv())
show(c1.recv())
show(c1.recv())
show(c1.recv())

//output:
//hello
//godevsig
//nihao
//123
//5.56
//"before close1"
//"before close2"
//<undefined>
//<undefined>
