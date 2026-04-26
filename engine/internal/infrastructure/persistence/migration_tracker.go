package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gorm.io/gorm"
)

const MigrationCollection = "ir_migration"

type MigrationRecord struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"size:255;not null;index"`
	Module    string    `gorm:"size:100;not null;index"`
	Batch     int       `gorm:"not null;index"`
	Model     string    `gorm:"size:255"`
	Source    string    `gorm:"size:50"`
	Records   int       `gorm:"default:0"`
	RecordIDs string    `gorm:"type:text"`
	Status    string    `gorm:"size:20;default:completed"`
	Error     string    `gorm:"type:text"`
	Duration  int64     `gorm:"default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (MigrationRecord) TableName() string {
	return MigrationCollection
}

func (r *MigrationRecord) GetRecordIDs() []string {
	if r.RecordIDs == "" {
		return nil
	}
	var ids []string
	json.Unmarshal([]byte(r.RecordIDs), &ids)
	return ids
}

func (r *MigrationRecord) SetRecordIDs(ids []string) {
	if len(ids) == 0 {
		r.RecordIDs = ""
		return
	}
	data, _ := json.Marshal(ids)
	r.RecordIDs = string(data)
}

type MigrationStatusEntry struct {
	Name     string
	Module   string
	Batch    int
	Model    string
	Source   string
	Records  int
	Status   string
	Duration string
	RanAt    string
}

type MigrationStore interface {
	HasRun(module, name string) bool
	NextBatch() int
	CurrentBatch() int
	Record(rec MigrationRecord) error
	RemoveByName(module, name string) error
	GetByBatch(batch int) ([]MigrationRecord, error)
	GetByModule(module string) ([]MigrationRecord, error)
	GetAll() ([]MigrationRecord, error)
	GetByName(module, name string) (*MigrationRecord, error)
	GetPending(module string, allNames []string) []string
	Reset(module string) error
	Status() ([]MigrationStatusEntry, error)
	Migrate() error
}

type GormMigrationStore struct {
	db *gorm.DB
}

func NewGormMigrationStore(db *gorm.DB) *GormMigrationStore {
	return &GormMigrationStore{db: db}
}

func (s *GormMigrationStore) Migrate() error {
	return s.db.AutoMigrate(&MigrationRecord{})
}

func (s *GormMigrationStore) HasRun(module, name string) bool {
	var count int64
	s.db.Model(&MigrationRecord{}).
		Where("module = ? AND name = ? AND status = ?", module, name, "completed").
		Count(&count)
	return count > 0
}

func (s *GormMigrationStore) NextBatch() int {
	var maxBatch int
	s.db.Model(&MigrationRecord{}).Select("COALESCE(MAX(batch), 0)").Scan(&maxBatch)
	return maxBatch + 1
}

func (s *GormMigrationStore) CurrentBatch() int {
	var maxBatch int
	s.db.Model(&MigrationRecord{}).Select("COALESCE(MAX(batch), 0)").Scan(&maxBatch)
	return maxBatch
}

func (s *GormMigrationStore) Record(rec MigrationRecord) error {
	return s.db.Create(&rec).Error
}

func (s *GormMigrationStore) RemoveByName(module, name string) error {
	return s.db.Where("module = ? AND name = ?", module, name).Delete(&MigrationRecord{}).Error
}

func (s *GormMigrationStore) GetByBatch(batch int) ([]MigrationRecord, error) {
	var records []MigrationRecord
	err := s.db.Where("batch = ?", batch).Order("id DESC").Find(&records).Error
	return records, err
}

func (s *GormMigrationStore) GetByModule(module string) ([]MigrationRecord, error) {
	var records []MigrationRecord
	err := s.db.Where("module = ?", module).Order("batch ASC, id ASC").Find(&records).Error
	return records, err
}

func (s *GormMigrationStore) GetAll() ([]MigrationRecord, error) {
	var records []MigrationRecord
	err := s.db.Order("batch ASC, id ASC").Find(&records).Error
	return records, err
}

func (s *GormMigrationStore) GetByName(module, name string) (*MigrationRecord, error) {
	var rec MigrationRecord
	err := s.db.Where("module = ? AND name = ? AND status = ?", module, name, "completed").First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *GormMigrationStore) GetPending(module string, allNames []string) []string {
	var ran []string
	s.db.Model(&MigrationRecord{}).
		Where("module = ? AND status = ?", module, "completed").
		Pluck("name", &ran)

	ranMap := make(map[string]bool, len(ran))
	for _, n := range ran {
		ranMap[n] = true
	}

	var pending []string
	for _, name := range allNames {
		if !ranMap[name] {
			pending = append(pending, name)
		}
	}
	return pending
}

func (s *GormMigrationStore) Reset(module string) error {
	if module == "" {
		return s.db.Where("1 = 1").Delete(&MigrationRecord{}).Error
	}
	return s.db.Where("module = ?", module).Delete(&MigrationRecord{}).Error
}

func (s *GormMigrationStore) Status() ([]MigrationStatusEntry, error) {
	var records []MigrationRecord
	if err := s.db.Order("batch ASC, id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return recordsToStatus(records), nil
}

type MongoMigrationStore struct {
	conn *MongoConnection
}

func NewMongoMigrationStore(conn *MongoConnection) *MongoMigrationStore {
	return &MongoMigrationStore{conn: conn}
}

func (s *MongoMigrationStore) Migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.conn.Database.CreateCollection(ctx, MigrationCollection)
	return nil
}

func (s *MongoMigrationStore) HasRun(module, name string) bool {
	ctx := context.Background()
	count, err := s.conn.Collection(MigrationCollection).CountDocuments(ctx, bson.M{
		"module": module, "name": name, "status": "completed",
	})
	return err == nil && count > 0
}

func (s *MongoMigrationStore) NextBatch() int  { return s.CurrentBatch() + 1 }

func (s *MongoMigrationStore) CurrentBatch() int {
	ctx := context.Background()
	opts := options.Find().SetSort(bson.D{{Key: "batch", Value: -1}}).SetLimit(1)
	cursor, err := s.conn.Collection(MigrationCollection).Find(ctx, bson.M{}, opts)
	if err != nil {
		return 0
	}
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			if b, ok := doc["batch"].(int32); ok {
				return int(b)
			}
			if b, ok := doc["batch"].(int64); ok {
				return int(b)
			}
		}
	}
	return 0
}

func (s *MongoMigrationStore) Record(rec MigrationRecord) error {
	ctx := context.Background()
	doc := bson.M{
		"_id":        fmt.Sprintf("%s:%s:%d", rec.Module, rec.Name, time.Now().UnixNano()),
		"name":       rec.Name,
		"module":     rec.Module,
		"batch":      rec.Batch,
		"model":      rec.Model,
		"source":     rec.Source,
		"records":    rec.Records,
		"record_ids": rec.RecordIDs,
		"status":     rec.Status,
		"error":      rec.Error,
		"duration":   rec.Duration,
		"created_at": rec.CreatedAt,
	}
	_, err := s.conn.Collection(MigrationCollection).InsertOne(ctx, doc)
	return err
}

func (s *MongoMigrationStore) RemoveByName(module, name string) error {
	ctx := context.Background()
	_, err := s.conn.Collection(MigrationCollection).DeleteMany(ctx, bson.M{"module": module, "name": name})
	return err
}

func (s *MongoMigrationStore) GetByBatch(batch int) ([]MigrationRecord, error) {
	ctx := context.Background()
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.conn.Collection(MigrationCollection).Find(ctx, bson.M{"batch": batch}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	return decodeMongoCursor(ctx, cursor)
}

func (s *MongoMigrationStore) GetByModule(module string) ([]MigrationRecord, error) {
	ctx := context.Background()
	opts := options.Find().SetSort(bson.D{{Key: "batch", Value: 1}, {Key: "created_at", Value: 1}})
	cursor, err := s.conn.Collection(MigrationCollection).Find(ctx, bson.M{"module": module}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	return decodeMongoCursor(ctx, cursor)
}

func (s *MongoMigrationStore) GetAll() ([]MigrationRecord, error) {
	ctx := context.Background()
	opts := options.Find().SetSort(bson.D{{Key: "batch", Value: 1}, {Key: "created_at", Value: 1}})
	cursor, err := s.conn.Collection(MigrationCollection).Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	return decodeMongoCursor(ctx, cursor)
}

func (s *MongoMigrationStore) GetByName(module, name string) (*MigrationRecord, error) {
	ctx := context.Background()
	var doc bson.M
	err := s.conn.Collection(MigrationCollection).FindOne(ctx, bson.M{
		"module": module, "name": name, "status": "completed",
	}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return bsonToMigrationRecord(doc), nil
}

func (s *MongoMigrationStore) GetPending(module string, allNames []string) []string {
	ctx := context.Background()
	cursor, err := s.conn.Collection(MigrationCollection).Find(ctx, bson.M{
		"module": module, "status": "completed",
	})
	if err != nil {
		return allNames
	}
	defer cursor.Close(ctx)

	ranMap := make(map[string]bool)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			if n, ok := doc["name"].(string); ok {
				ranMap[n] = true
			}
		}
	}

	var pending []string
	for _, name := range allNames {
		if !ranMap[name] {
			pending = append(pending, name)
		}
	}
	return pending
}

func (s *MongoMigrationStore) Reset(module string) error {
	ctx := context.Background()
	filter := bson.M{}
	if module != "" {
		filter["module"] = module
	}
	_, err := s.conn.Collection(MigrationCollection).DeleteMany(ctx, filter)
	return err
}

func (s *MongoMigrationStore) Status() ([]MigrationStatusEntry, error) {
	records, err := s.GetAll()
	if err != nil {
		return nil, err
	}
	return recordsToStatus(records), nil
}

func recordsToStatus(records []MigrationRecord) []MigrationStatusEntry {
	entries := make([]MigrationStatusEntry, len(records))
	for i, r := range records {
		entries[i] = MigrationStatusEntry{
			Name: r.Name, Module: r.Module, Batch: r.Batch,
			Model: r.Model, Source: r.Source, Records: r.Records,
			Status: r.Status, Duration: fmt.Sprintf("%dms", r.Duration),
			RanAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return entries
}

func decodeMongoCursor(ctx context.Context, cursor interface{ Next(context.Context) bool; Decode(interface{}) error }) ([]MigrationRecord, error) {
	var records []MigrationRecord
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		records = append(records, *bsonToMigrationRecord(doc))
	}
	return records, nil
}

func bsonToMigrationRecord(doc bson.M) *MigrationRecord {
	rec := &MigrationRecord{
		Name: bsonStr(doc, "name"), Module: bsonStr(doc, "module"),
		Model: bsonStr(doc, "model"), Source: bsonStr(doc, "source"),
		Status: bsonStr(doc, "status"), Error: bsonStr(doc, "error"),
		RecordIDs: bsonStr(doc, "record_ids"),
	}
	if b, ok := doc["batch"].(int32); ok { rec.Batch = int(b) } else if b, ok := doc["batch"].(int64); ok { rec.Batch = int(b) }
	if r, ok := doc["records"].(int32); ok { rec.Records = int(r) } else if r, ok := doc["records"].(int64); ok { rec.Records = int(r) }
	if d, ok := doc["duration"].(int64); ok { rec.Duration = d } else if d, ok := doc["duration"].(int32); ok { rec.Duration = int64(d) }
	if t, ok := doc["created_at"].(time.Time); ok { rec.CreatedAt = t }
	return rec
}

func bsonStr(doc bson.M, key string) string {
	if v, ok := doc[key].(string); ok {
		return v
	}
	return ""
}

type MigrationTracker struct {
	Store MigrationStore
}

func NewMigrationTracker(db *gorm.DB) *MigrationTracker {
	return &MigrationTracker{Store: NewGormMigrationStore(db)}
}

func NewMigrationTrackerFromStore(store MigrationStore) *MigrationTracker {
	return &MigrationTracker{Store: store}
}

func AutoMigrateMigrationTracker(db *gorm.DB) error {
	return NewGormMigrationStore(db).Migrate()
}

func (t *MigrationTracker) HasRun(module, name string) bool        { return t.Store.HasRun(module, name) }
func (t *MigrationTracker) NextBatch() int                         { return t.Store.NextBatch() }
func (t *MigrationTracker) CurrentBatch() int                      { return t.Store.CurrentBatch() }
func (t *MigrationTracker) RemoveByName(module, name string) error { return t.Store.RemoveByName(module, name) }
func (t *MigrationTracker) GetByBatch(batch int) ([]MigrationRecord, error) { return t.Store.GetByBatch(batch) }
func (t *MigrationTracker) GetByModule(module string) ([]MigrationRecord, error) { return t.Store.GetByModule(module) }
func (t *MigrationTracker) GetAll() ([]MigrationRecord, error)     { return t.Store.GetAll() }
func (t *MigrationTracker) GetByName(module, name string) (*MigrationRecord, error) { return t.Store.GetByName(module, name) }
func (t *MigrationTracker) GetPending(module string, allNames []string) []string { return t.Store.GetPending(module, allNames) }
func (t *MigrationTracker) Reset(module string) error               { return t.Store.Reset(module) }
func (t *MigrationTracker) Status() ([]MigrationStatusEntry, error) { return t.Store.Status() }

func (t *MigrationTracker) Record(module, name, model, source string, records int, batch int, duration time.Duration, recordIDs []string, err error) error {
	rec := MigrationRecord{
		Name: name, Module: module, Batch: batch, Model: model,
		Source: source, Records: records, Status: "completed",
		Duration: duration.Milliseconds(), CreatedAt: time.Now(),
	}
	rec.SetRecordIDs(recordIDs)
	if err != nil {
		rec.Status = "failed"
		rec.Error = err.Error()
	}
	return t.Store.Record(rec)
}
