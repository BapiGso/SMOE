package handler

import (
	"SMOE/moe/store"
)

func sortComms(data []store.Comments) [][]store.Comments {
	var final [][]store.Comments
	parentMap := make(map[uint]int)
	for _, v := range data {
		//父评论新建一个组，因为按时间排序肯定比子评论先
		if v.Parent == 0 {
			//初始化tmp的同时就把v加入切片
			tmp := []store.Comments{v}
			final = append(final, tmp)
			parentMap[v.Coid] = len(final) - 1
		} else { //如果是子评论，找自己属于哪个父评论组
			index := parentMap[v.Parent]
			final[index] = append(final[index], v)
			parentMap[v.Coid] = index
		}
	}
	return final
}
