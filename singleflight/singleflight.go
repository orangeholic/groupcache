/*
Copyright 2012 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package singleflight provides a duplicate function call suppression
// mechanism.
package singleflight

import "sync"

// call is an in-flight or completed Do call
/*存储一条命令*/
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m/*保护m单线程访问*/
	m  map[string]*call // lazily initialized/*一组同样的命令  一个string表示key 这个key代表的命名有同样的逻辑和返回*/
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()/*所有的同key的命令全部等待第一个的执行完成*/
		return c.val, c.err/*所有同key的命令返回第一个命令执行的结果*/
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c/*第一个命令把自己加入到group的m中，并通过wg阻挡以后进来的同key命令*/
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()/*请他命令可以返回了，第一个命令还要把自己从group的m中删除*/

	g.mu.Lock()
	delete(g.m, key)/*这同key的命令全部执行返回了，从group的m中删除key*/
	g.mu.Unlock()

	return c.val, c.err
}
