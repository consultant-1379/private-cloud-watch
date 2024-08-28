package lib

import (
	"fmt"
	"strings"
)

// FSRouter executes a router. normally, we'd use it as go FSRouter()
func FSRouter(cmd chan FSRouterCmd, input chan EventFS, finished chan bool) {
	dir := make(map[string][]chan<- EventFS)
	prefix := make(map[string][]chan<- EventFS)
	id := make(map[string][]chan<- EventFS)

	proc := func(i EventFS) {
		if i.Op == FSEexecstatus {
			fmt.Printf("==== rexmit %s\n", i.Path)
			if clist, ok := id[i.Path]; ok {
				fmt.Printf("==== xmiting %s on\n", i.Path)
				for _, ch := range clist {
					ch <- i
				}
			}
			return
		}
		if clist, ok := dir[i.Path]; ok {
			fmt.Printf("==== sending %s on\n", i.Path)
			for _, ch := range clist {
				ch <- i
			}
		}
		for x, clist := range prefix {
			if strings.HasPrefix(i.Path, x) {
				for _, ch := range clist {
					ch <- i
				}
			}
		}
	}

	// process events and cmds intermixed
normalLoop:
	for {
		select {
		case c := <-cmd:
			switch c.Op {
			case FSRopen:
				fmt.Printf("registering %v\n", c.Files)
				for _, f := range c.Files {
					fsrPlus(dir, c.Dest, f)
				}
				if c.ID != "" {
					fsrPlus(id, c.Dest, c.ID)
					fmt.Printf("registering id %s\n", c.ID)
				}
			case FSRprefix:
				for _, f := range c.Files {
					fsrPlus(prefix, c.Dest, f)
				}
				if c.ID != "" {
					fsrPlus(id, c.Dest, c.ID)
				}
			case FSRclose:
				fsrMinus(dir, c.Dest)
				if c.ID != "" {
					fsrMinus(id, c.Dest)
				}
			case FSRexit:
				break normalLoop
			}
		case ix := <-input:
			proc(ix)
		}
	}
	// done; now just drain the events
drainLoop:
	for {
		select {
		case i := <-input:
			proc(i)
		default:
			break drainLoop
		}
	}
	finished <- true
}

func fsrPlus(dir map[string][]chan<- EventFS, ch chan<- EventFS, path string) {
	if list, ok := dir[path]; ok {
		dir[path] = append(list, ch)
	} else {
		var list []chan<- EventFS
		list = append(list, ch)
		dir[path] = list
	}
}

func fsrMinus(dir map[string][]chan<- EventFS, ch chan<- EventFS) {
	for k, v := range dir {
		for i := range v {
			if v[i] == ch {
				if len(v) == 1 {
					delete(dir, k)
				} else {
					switch {
					case i == 0:
						dir[k] = v[1:]
					case i == len(v)-1:
						dir[k] = v[:i-1]
					default:
						xx := v[0 : i-1]
						dir[k] = append(xx, v[i+1:]...)
					}
				}
				break
			}
		}
	}
}
