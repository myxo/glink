package glink

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const db_path = "tmp.db"

func TestDbWriteCid(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)
	cid1 := db.GetOwnInfo().Uid

	db, err = NewDb(db_path)
	assert.Nil(err)
	cid2 := db.GetOwnInfo().Uid

	os.Remove(db_path)
	assert.Equal(cid1, cid2)
}

func TestDbWriteName(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)
	err = db.SetOwnUid("uid")
	assert.Nil(err)

	err = db.SetOwnName("my_name")
	assert.Nil(err)

	name1 := db.GetOwnInfo().Name
	assert.NotEqual(name1, "")

	db, err = NewDb(db_path)
	assert.Nil(err)
	name2 := db.GetOwnInfo().Name

	os.Remove(db_path)
	assert.Equal(name1, name2)
}

func TestDbWriteMsg(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)
	db, err := NewDb(db_path)
	assert.Nil(err)

	from := Uid("from_cid")
	to := Cid("to_cid")

	msg1 := ChatMessage{Uid: from, Cid: to, Index: 1, Text: "test text"}
	msg2 := ChatMessage{Uid: from + "2", Cid: to, Index: 1, Text: "test text 2"}
	err = db.SaveMessage(msg1)
	assert.Nil(err)
	err = db.SaveMessage(msg2)
	assert.Nil(err)

	msgs, err := db.GetMessages(to, 0, 100000)
	assert.Nil(err)
	assert.Equal(len(msgs), 2)
	assert.Equal([]ChatMessage{msg1, msg2}, msgs)

	db, err = NewDb(db_path)
	assert.Nil(err)

	msgs, err = db.GetMessages(to, 0, 100000)
	assert.Nil(err)
	assert.Equal(len(msgs), 2)
	assert.Equal([]ChatMessage{msg1, msg2}, msgs)
}

func TestDbWriteMsgIndex(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)
	db, err := NewDb(db_path)
	assert.Nil(err)

	from := Uid("from_cid")
	to := Cid("to_cid")

	msg := ChatMessage{Uid: from, Cid: to, Index: 1, Text: "test text"}
	err = db.SaveMessage(msg)
	assert.Nil(err)
	msg = ChatMessage{Uid: from, Cid: to, Index: 2, Text: "test text 2"}
	err = db.SaveMessage(msg)
	assert.Nil(err)
	msg = ChatMessage{Uid: from, Cid: to, Index: 3, Text: "test text 3"}
	err = db.SaveMessage(msg)
	assert.Nil(err)

	msgs, err := db.GetMessages(to, 2, 10)
	assert.Nil(err)
	assert.Equal(len(msgs), 2)
	assert.Equal("test text 2", msgs[0].Text)
	assert.Equal("test text 3", msgs[1].Text)

	os.Remove(db_path)
}

func TestDbHasUid(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)

	db.SaveNewUid("uid1", "name", "")

	assert.True(db.IsKnownUid("uid1"))
	assert.False(db.IsKnownUid("uid2"))

	os.Remove(db_path)
}

func TestVectorClock(t *testing.T) {
	assert := assert.New(t)
	db, err := NewDb("")
	assert.Nil(err)

	err = db.SaveMessage(ChatMessage{Uid: "uid1", Cid: "cid1", Index: 1, Text: "test"})
	assert.Nil(err)
	err = db.SaveMessage(ChatMessage{Uid: "uid1", Cid: "cid1", Index: 2, Text: "test"})
	assert.Nil(err)
	err = db.SaveMessage(ChatMessage{Uid: "uid2", Cid: "cid1", Index: 1, Text: "test"})
	assert.Nil(err)

	err = db.SaveMessage(ChatMessage{Uid: "uid1", Cid: "cid2", Index: 1, Text: "test"})
	assert.Nil(err)
	
	err = db.SaveMessage(ChatMessage{Uid: "uid1", Cid: "cid3", Index: 1, Text: "test"})
	assert.Nil(err)


	expected := map[Cid]VectorClock{
		"cid1": {
			"uid1": 2, 
			"uid2": 1,
		},
		"cid2": {
			"uid1": 1,
		},
	}
	vc, err := db.GetVectorClockOfCids([]Cid{"cid1", "cid2"})
	assert.Nil(err)
	require.Equal(t, expected, vc)
}
