package khan

import (
	"math"

	kd "github.com/erixzone/crux/pkg/khan/defn"
)

/*
	strew generates all the possible solution sets for picking n instances out
	of a given population. ordinarily, this is just brute force, but it gets tricky when
	n > len(pop).

	keep in mind that a solution set is a distribution (nodes+counts), not just a node set.
*/
func strew(n int, pop []int) [][]int {
	if len(pop) == 0 {
		return nil
	}
	k := int(math.Ceil(float64(n) / float64(len(pop))))
	var ret [][]int
	temp := make([]int, kd.SetMax(pop, nil)+1)
	ret = strew1(n, pop, k, ret, 0, temp, 0)
	//	fmt.Printf("strew(%d from %v) returns %v\n", n, pop, ret)
	return ret
}

func strew1(n int, pop []int, max int, ans [][]int, index int, temp []int, tempCount int) [][]int {
	me := pop[index]
	index++
	if tempCount+max > n {
		max = n - tempCount
	}
	// creat a copy of temp so we can undo recursion easily
	tempc := make([]int, len(temp))
	copy(tempc, temp)
	for j := 0; j <= max; j++ {
		temp[me] = j
		if (tempCount + j) == n {
			tempx := make([]int, len(temp))
			copy(tempx, temp)
			ans = append(ans, tempx)
		} else {
			if index < len(pop) {
				ans = strew1(n, pop, max, ans, index, temp, tempCount+j)
				copy(temp, tempc)
			}
		}
	}
	return ans
}
