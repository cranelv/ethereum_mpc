package rlp

import (
	"testing"
	"sync"
)

type testInterface interface {
	test1()
	test2()
	test3()
	GetType()uint16
}
type testStruct1 struct {
	A uint64
	B uint64
	C uint64
}
func (t *testStruct1)test1(){

}
func (t *testStruct1)test2(){

}
func (t *testStruct1)test3(){

}
func (t *testStruct1)GetType()uint16{
	return 10
}
type testStruct2 struct {
	A uint64
	B uint64
	C uint64
	D uint64
}
func (t *testStruct2)test1(){

}
func (t *testStruct2)test2(){

}
func (t *testStruct2)test3(){

}
func (t *testStruct2)GetType()uint16{
	return 20
}
type testStruct struct {
	Test1 testInterface //`rlp:"interface"`
	Test2 testInterface //`rlp:"interface"`
//	Test3
}
func TestDecodeInterface1(t *testing.T) {
	testRlp := testStruct{&testStruct1{100,100,100},&testStruct2{100,100,100,100}}
	b,_ := EncodeToBytes(testRlp)
	t.Log(b)
	testRlp1 := testStruct{}
//	testSlice1 := []testInterface{}
	interfaceConstructorMap[testRlp.Test1.GetType()] = func()interface{}{
		return &testStruct1{}
	}
	interfaceConstructorMap[testRlp.Test2.GetType()] = func()interface{}{
		return &testStruct2{}
	}
	DecodeBytes(b,&testRlp1)
	t.Log(testRlp1.Test1,testRlp1.Test2)
}
func TestDecodeInterface(t *testing.T) {
	test1 := testStruct1{100,100,100}
	interfaceConstructorMap[test1.GetType()] = func()interface{}{
		return &testStruct1{}
	}
	test2 := testStruct2{100,100,100,100}
	interfaceConstructorMap[test2.GetType()] = func()interface{}{
		return &testStruct2{}
	}
	wg := sync.WaitGroup{}
	for i:=0;i<10000;i++{
		wg.Add(1)
		go func() {
			testSlice := []testInterface{}
			test1 := testStruct1{100,100,100}
			testSlice = append(testSlice,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&test1,&testStruct1{100,100,100},&testStruct1{200,200,200})
			testSlice = append(testSlice,&testStruct2{100,100,100,100},&testStruct2{100,100,100,100})

			b1,_ := EncodeToBytes(test1)
			b,_ := EncodeToBytes(testSlice)
			testSlice1 := []testInterface{}
			DecodeBytes(b,&testSlice1)
			DecodeBytes(b1,test1)
			t.Log(testSlice1[0],testSlice1[1],testSlice1[2],testSlice1[3])
			t.Log(test1)
			wg.Done()
		}()
	}
	wg.Wait()
}
