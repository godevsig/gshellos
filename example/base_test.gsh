fmt := import("fmt")
os := import("os")
times := import("times")

fmt.println("nihao")
fmt.println(os.args())

times.sleep(30*times.second)

each := func(seq, fn) {
    for x in seq { fn(x) }
}

sum := func(init, seq) {
    each(seq, func(x) { init += x })
    return init
}

//len("123", "567")

fmt.println(len("hello"))
times.sleep(30*times.second)

times.sleep(1*times.second)
fmt.println(sum(0, [1, 2, 3]))   // "6"
fmt.println(sum("", [1, 2, 3]))  // "123"

//output:
//5
//6
//123
