package service

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"matrixchaindata/utils"
)

// 根据区块hash, 或者是高度查早到区块信息
//where := bson.M{[{"blockid": "block_hash"},{'_id': block_height}]}
//
//cursor, err := coll.Find(
//	context.Background(),
//	bson.D{
//		{"status", "A"},
//		{"$or", bson.A{
//			bson.D{{"qty", bson.D{{"$lt", 30}}}},
//			bson.D{{"item", primitive.Regex{Pattern: "^p", Options: ""}}},
//		}},
//	})
//for cursor.Next(context.TODO()) {
//	elem := &bson.D{}
//	if err := cursor.Decode(elem); err != nil {
//		log.Fatal(err)
//	}
//	// ideally, you would do something with elem....
//	// but for now just print it to the console
//	fmt.Println(elem)
//}
// 区块信息
func (s *Serve) GetBlock(block_hash string, block_height int64, bcname string) (bson.M, error) {
	elem := bson.M{}
	err := s.Dao.MongoClient.Collection(utils.BlockCol(bcname)).FindOne(
		nil,
		bson.D{{"$or", bson.A{
			bson.D{{"blockid", block_hash}},
			bson.D{{"_id", block_height}},
		}},
		}).Decode(&elem)
	if err != nil {
		return nil, err
	}
	return elem, nil
}

// 链的区块高度（当前链的高度）
func (s *Serve) GetBlockCount(bcname string) (int64, error) {

	//cursor, err := s.Dao.MongoClient.Collection(utils.BlockCol(bcname)).Find(nil,bson.D{})
	//if err != nil {
	//	return 0, err
	//}
	//result := []bson.M{}
	//cursor.All(nil,&result)
	return s.Dao.MongoClient.Collection(utils.BlockCol(bcname)).CountDocuments(nil, bson.D{}, options.Count())

	//return int64(len(result)), nil
}

// 获取区块列表
// 以高度作为开始下标获取指定条数的区块信息
func (s *Serve) GetBockekList(height int64, num int64, bcname string) ([]bson.M, error) {
	opts := options.Find()
	opts.SetSort(bson.D{{"_id", 1}})
	opts.SetLimit(num)

	cursor, err := s.Dao.MongoClient.Collection(utils.BlockCol(bcname)).Find(
		nil,
		bson.M{"_id": bson.M{"$gte": height}},
		opts)

	if err != nil {
		return nil, err
	}
	elems := []bson.M{}
	cursor.All(nil, &elems)
	return elems, nil
}
