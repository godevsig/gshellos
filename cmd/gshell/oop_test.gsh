fmt := import("fmt")

// object
newCgroup := func(name) {
    cg := {name:name, pids:[]}

    cg.addPid = func(pid) {
        fmt.println("add ", pid, " to cgroup ", cg.name)
        cg.pids = append(cg.pids, pid)
        fmt.println("all pids:", cg.pids)
    }

    return cg
}

// object inheritance
newCgroupExt := func(name, mode) {
    cg := newCgroup(name)
    cg.mode = mode

    cg.showMode = func() {
        fmt.println(cg.mode)
    }

    return cg
}

cg1 := newCgroup("cg1")
cg2 := newCgroup("cg2")

cg1.addPid(123)
cg1.addPid(456)

cg2.addPid(111)
cg2.addPid(222)
cg2.addPid(333)

cg1.addPid(789)

cgext := newCgroupExt("cgext", "thread")
cgext.addPid(888)
cgext.showMode()

//output:
//add 123 to cgroup cg1
//all pids:[123]
//add 456 to cgroup cg1
//all pids:[123, 456]
//add 111 to cgroup cg2
//all pids:[111]
//add 222 to cgroup cg2
//all pids:[111, 222]
//add 333 to cgroup cg2
//all pids:[111, 222, 333]
//add 789 to cgroup cg1
//all pids:[123, 456, 789]
//add 888 to cgroup cgext
//all pids:[888]
//thread
