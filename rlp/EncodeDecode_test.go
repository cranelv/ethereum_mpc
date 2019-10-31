package rlp

import (
	"testing"
	"bytes"
	"io"
	"sync"
	"math/rand"
)
type testBytes struct {
	A,B,C []byte
}
type testByte struct {
	A,B,C byte
}
type testUint struct {
	A,B,C uint8
}
type mutexMap struct{
	mu sync.Mutex
	data map[int]string
}
func TestMutexMap(t *testing.T){
	wg := sync.WaitGroup{}
	mm := mutexMap{data:make(map[int]string)}
	for i:=0;i<100;i++{
		mm.data[i] = "test1111"
	}
	for i:=0;i<100;i++{
		wg.Add(1)
		go func() {
			for j:=0;j<1e10;j++{
				mm.mu.Lock()
				mm.data[rand.Intn(100000)] = "test222222"
				mm.mu.Unlock()
			}
			wg.Done()
		}()
	}
	for i:=0;i<100;i++{
		wg.Add(1)
		go func(index int) {
			for j:=0;j<1e10;j++{
				mm.mu.Lock()
				data1 := mm.data
				mm.mu.Unlock()
				value,exist := data1[i]
				if !exist{
					t.Error("Map Exsit False")
				}else{
					t.Log(value)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}
func TestEncodeBytes(t *testing.T) {
	test1 := testBytes{[]byte{0},[]byte{1},[]byte{2}}
	buff1,_ := EncodeToBytes(test1)
	t.Log(buff1)
	test2 := testByte{0,1,2}
	buff2,_ := EncodeToBytes(test2)
	t.Log(buff2)
	test3 := testUint{0,1,2}
	buff3,_ := EncodeToBytes(test3)
	t.Log(buff3)
}
func TestEncodeItems(t *testing.T) {
	test1 := testStruct1{100,200,300}
	var buffer []byte
	writer := bytes.NewBuffer(buffer)
	err:= Encode(writer,&test1)
	if err != nil{
		t.Error(err)
	}
	Encode(writer,&test1)
	Encode(writer,&test1)
	Encode(writer,&test1)
	t.Log(writer.Bytes())
}
func TestEncodeDecode(t *testing.T) {
	test1 := testStruct1{100,200,300}
	var buffer []byte
	writer := bytes.NewBuffer(buffer)
	err:= Encode(writer,&test1)
	if err != nil{
		t.Error(err)
	}
	Encode(writer,&testStruct2{400,200,300,400})
	Encode(writer,&testStruct1{500,200,300})
	Encode(writer,&testStruct1{600,200,300})
	t.Log(writer.Bytes())
	for i:=0;i<5;i++{
		reader := bytes.NewReader(writer.Bytes())
		var test2 testStruct1
		err = DecodeTupleItem(reader,i,&test2)
		if err != nil{
			t.Error(err)
		}else{
			t.Log(test2)
		}
	}

}
type testRlp struct {
	A []uint
	B []byte
	C string
}
func (ts *testRlp)EncodeRLP(w io.Writer) error {
	Encode(w,ts.A)
	if len(ts.A) > 0{
		return Encode(w,ts.B)
	}else{
		return Encode(w,ts.C)
	}
}
func (ts *testRlp)DecodeRLP(s *Stream) error {
	s.Decode(&ts.A)
	if len(ts.A) > 0{
		return s.Decode(&ts.B)
	}else{
		return s.Decode(&ts.C)
	}
}
type testRlpGroup struct{
	AAA []uint
	AA testRlp
	BB testRlp
	CCC string
}
func TestEncodeDecodeList(t *testing.T){
	testGroup := testRlpGroup{[]uint{100,2000,300,400},
	testRlp{[]uint{},[]byte{99,99,99,99},"testAAAAAAAAAAAA"},
	testRlp{[]uint{100,200,300},[]byte{88,88,88,88},"testBBBBBBBBBBBB"},
	"CCCCCCCCCCCCCCCCC"}
	var buffer []byte
	writer := bytes.NewBuffer(buffer)
	err:= Encode(writer,&testGroup)
	if err != nil{
		t.Error(err)
	}
	testGroup1 := &testRlpGroup{}
	reader := bytes.NewReader(writer.Bytes())
	Decode(reader,testGroup1)
	t.Log(testGroup1)
}