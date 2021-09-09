fmt := import("fmt")
os := import("os")

fmt.println("nihao")
fmt.println(os.args())

each := func(seq, fn) {
    for x in seq { fn(x) }
}

sum := func(init, seq) {
    each(seq, func(x) { init += x })
    return init
}

fmt.println(len("hello"))
fmt.println(sum(0, [1, 2, 3]))   // "6"
fmt.println(sum("", [1, 2, 3]))  // "123"

//output:
//5
//6
//123
