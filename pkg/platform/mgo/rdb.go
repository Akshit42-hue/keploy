package mgo

import (
	"context"
	"time"

	"go.keploy.io/server/pkg/service/run"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func NewRun(c *mongo.Collection, test *mongo.Collection, log *zap.Logger) *RunDB {
	return &RunDB{
		c:    c,
		log:  log,
		test: test,
	}
}

type RunDB struct {
	c    *mongo.Collection
	test *mongo.Collection
	log  *zap.Logger
}

func (r *RunDB) ReadTest(ctx context.Context, id string) (run.Test, error) {
	// too repetitive
	// TODO write a generic FindOne for all get calls
	filter := bson.M{"_id": id}
	var t run.Test
	err := r.c.FindOne(ctx, filter).Decode(&t)
	if err != nil {
		return t, err
	}
	return t, nil
}

func (r *RunDB) ReadTests(ctx context.Context, runID string) ([]run.Test, error) {
	filter := bson.M{"run_id": runID}
	findOptions := options.Find()

	var res []run.Test
	cur, err := r.test.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	// Loop through the cursor
	for cur.Next(ctx) {
		var t run.Test
		err = cur.Decode(&t)
		if err != nil {
			return nil, err

		}
		res = append(res, t)
	}

	if err = cur.Err(); err != nil {
		return nil, err

	}

	err = cur.Close(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *RunDB) PutTest(ctx context.Context, t run.Test) error {
	upsert := true
	opt := &options.UpdateOptions{
		Upsert: &upsert,
	}
	filter := bson.M{"_id": t.ID}
	update := bson.D{{"$set", t}}

	_, err := r.test.UpdateOne(ctx, filter, update, opt)
	if err != nil {
		//t.log.Error("failed to insert testcase into DB", zap.String("cid", tc.CID), zap.String("appid", tc.AppID), zap.String("id", tc.ID), zap.Error())
		return err
	}
	return nil
}

func (r *RunDB) Read(ctx context.Context, cid string, user, app, id *string, from, to *time.Time, offset *int, limit *int) ([]*run.TestRun, error) {
	off, lim := 0, 25
	if offset != nil {
		off = *offset
	}
	if limit == nil {
		lim = *limit
	}

	filter := bson.M{
		"cid": cid,
	}
	if user != nil {
		filter["user"] = user
	}
	if app != nil {
		filter["app"] = app
	}
	if id != nil {
		filter["_id"] = id
	}

	if from != nil {
		filter["updated"] = bson.M{"$gte": from.Unix()}
	}

	if to != nil {
		filter["updated"] = bson.M{"$lte": to.Unix()}
	}

	var tcs []*run.TestRun
	options := options.Find()
	if off > 0 {
		off = ((off - 1) * lim)
	} else {
		off = 0
	}
	options.SetSort(bson.M{"$lte": to.Unix()})
	options.SetSkip(int64(off))
	options.SetLimit(int64(lim))

	cur, err := r.c.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}

	// Loop through the cursor
	for cur.Next(ctx) {
		var tc *run.TestRun
		err = cur.Decode(&tc)
		if err != nil {
			return nil, err

		}
		tcs = append(tcs, tc)
	}

	if err = cur.Err(); err != nil {
		return nil, err

	}

	err = cur.Close(ctx)
	if err != nil {
		return nil, err
	}
	return tcs, nil
}

func (r *RunDB) Upsert(ctx context.Context, run run.TestRun) error {
	upsert := true
	opt := &options.UpdateOptions{
		Upsert: &upsert,
	}
	filter := bson.M{"_id": run.ID}
	update := bson.D{{"$set", run}}

	_, err := r.c.UpdateOne(ctx, filter, update, opt)
	if err != nil {
		//t.log.Error("failed to insert testcase into DB", zap.String("cid", tc.CID), zap.String("appid", tc.AppID), zap.String("id", tc.ID), zap.Error())
		return err
	}
	return nil
}

func (r *RunDB) Increment(ctx context.Context, success, failure bool, id string) error {
	update := bson.M{}
	if success {
		update["$inc"] = bson.D{{"success", 1}}
	}
	if failure {
		update["$inc"] = bson.D{{"failure", 1}}
	}

	_, err := r.c.UpdateOne(ctx, bson.M{
		"_id": id,
	}, update, options.Update().SetUpsert(true))

	if err != nil {
		return err
	}
	return nil
}
