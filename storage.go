// storage.go
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/postgresql"
)

type Grid struct {
	ID            int64      `db:"id,omitempty"`
	Symbol        string     `db:"symbol"`
	StartTime     time.Time  `db:"start_time"`
	StartPrice    float64    `db:"start_price"`
	OpenQuantity  float64    `db:"open_quantity"`
	OpenAmount    float64    `db:"open_amount"`
	OpenOrderID   int64      `db:"open_order_id"`
	EndTime       *time.Time `db:"end_time,omitempty"`
	EndPrice      *float64   `db:"end_price,omitempty"`
	Status        string     `db:"status"`
	CloseQuantity *float64   `db:"close_quantity,omitempty"`
	CloseAmount   *float64   `db:"close_amount,omitempty"`
	CloseOrderID  *int64     `db:"close_order_id,omitempty"`
	Profit        *float64   `db:"profit,omitempty"`
	Fee           *float64   `db:"fee,omitempty"`
	Level         int        `db:"current_level"` // 当前网格级别
}

func NewGrid(symbol string, startPrice float64, openSize float64, openId int64, level int) Grid {
	return Grid{Symbol: symbol, StartTime: time.Now(), StartPrice: startPrice, OpenQuantity: openSize, OpenAmount: math.Abs(startPrice * openSize), OpenOrderID: openId, Status: RunningStatus, Level: level}
}

// Orders 表格结构（状态更新为 Init, Opened, Completed）
type Orders struct {
	ID       int64     `db:"id,omitempty"`
	GridID   int64     `db:"grid_id"`
	Level    int       `db:"level"`
	Price    float64   `db:"price"`
	Quantity float64   `db:"quantity"`
	Amount   float64   `db:"amount"`
	Time     time.Time `db:"open_time"`
	OrderID  int64     `db:"order_id"` // 开仓订单ID
	Side     string    `db:"side"`
	Fee      float64   `db:"fee"`
	Status   string    `db:"status"` // Init, Opened, Completed
}

type Storage struct {
	db db.Session
}

func NewStorage(connString string) (*Storage, error) {
	settings, err := postgresql.ParseURL(connString)
	if err != nil {
		return nil, err
	}

	sess, err := postgresql.Open(settings)
	if err != nil {
		return nil, err
	}

	err = setupTables(sess)
	if err != nil {
		log.Println(err)
		err := sess.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	return &Storage{db: sess}, nil
}

func setupTables(sess db.Session) error {
	_, err := sess.SQL().Exec(`
        CREATE TABLE IF NOT EXISTS grid (
            id SERIAL PRIMARY KEY,
            symbol VARCHAR(20) NOT NULL,
            start_time TIMESTAMP NOT NULL,
            start_price DOUBLE PRECISION NOT NULL,
            open_quantity DOUBLE PRECISION NOT NULL,
            open_amount DOUBLE PRECISION NOT NULL,
            open_order_id bigint NOT NULL,
            end_time TIMESTAMP,
            end_price DOUBLE PRECISION,
            status VARCHAR(20) NOT NULL CHECK (status IN ('Running', 'Terminated')),
            close_quantity DOUBLE PRECISION,
            close_amount DOUBLE PRECISION,
            close_order_id bigint,
            profit DOUBLE PRECISION,
            fee DOUBLE PRECISION,
            current_level INT NOT NULL DEFAULT 0
        )
    `)
	if err != nil {
		return err
	}

	_, err = sess.SQL().Exec(`
        CREATE TABLE IF NOT EXISTS orders (
            id SERIAL PRIMARY KEY,
            grid_id BIGINT NOT NULL REFERENCES grid(id),
            level INT NOT NULL,
            price DOUBLE PRECISION NOT NULL,
            quantity DOUBLE PRECISION NOT NULL,
            amount DOUBLE PRECISION NOT NULL,
            open_time TIMESTAMP NOT NULL,
            order_id bigint NOT NULL,
            side VARCHAR(20) NOT NULL,
            fee DOUBLE PRECISION,
            status VARCHAR(20) NOT NULL CHECK (status IN ('Init',  'Completed','Cancelled'))
        )
    `)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) Close() {
	s.db.Close()
}
func (s *Storage) GetGridByGridId(id int64) Grid {
	var grid Grid
	err := s.db.Collection("grid").Find(db.Cond{"id": id}).One(&grid)
	if err != nil {
		log.Println("GetGridByGridByGridId:", err)
		return Grid{}
	}
	return grid
}
func (s *Storage) GetAllGrids() []Grid {
	var grids []Grid
	err := s.db.Collection("grid").Find().All(&grids)
	if err != nil {
		log.Println("GetAllGrids:", err)
		return []Grid{}
	}
	return grids
}
func (s *Storage) InsertGrid(grid Grid) (int64, error) {

	count, err := s.db.Collection("grid").
		Find(db.Cond{"symbol": grid.Symbol, "status": RunningStatus}).Count()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, fmt.Errorf("a running grid already exists for symbol %s", grid.Symbol)
	}

	res, err := s.db.Collection("grid").Insert(grid)
	if err != nil {
		return 0, err
	}
	id := res.ID().(int64)
	grid.ID = id
	return id, nil
}

func (s *Storage) UpdateGrid(id int64, grid Grid) error {
	return s.db.Collection("grid").Find(db.Cond{"id": id}).Update(grid)
}

func (s *Storage) UpdateGridLevel(id int64, level int) error {
	var grid Grid
	e := s.db.Collection("grid").Find(db.Cond{"id": id}).One(&grid)
	if e != nil {
		return e
	}
	grid.Level = level
	return s.UpdateGrid(id, grid)

}

func (s *Storage) GetRunningGridBySymbol(symbol string) (*Grid, error) {
	var grid Grid
	err := s.db.Collection("grid").
		Find(db.Cond{"symbol": symbol, "status": RunningStatus}).
		One(&grid)
	if err != nil {
		return nil, err
	}
	return &grid, nil
}

func (s *Storage) InsertOrder(order Orders) (int64, error) {
	res, err := s.db.Collection("orders").Insert(order)
	if err != nil {
		return 0, err
	}
	id := res.ID().(int64)
	return id, nil
}

func (s *Storage) UpdateOrder(id int64, order Orders) error {
	return s.db.Collection("orders").Find(db.Cond{"id": id}).Update(order)
}

func (s *Storage) UpdateOrderStatus(oid int64, oldStatus, status string) error {
	var order Orders
	e := s.db.Collection("orders").Find(db.Cond{"status": oldStatus, "order_id": oid}).One(&order)
	if e != nil {
		return e
	}
	order.Status = status
	return s.UpdateOrder(order.ID, order)
}
func (s *Storage) CompletedOrders(gridID int64, side string, levelDesc bool) ([]Orders, error) {
	var orders []Orders
	orderBy := "level"
	if levelDesc {
		orderBy = "-level"
	}
	err := s.db.Collection("orders").
		Find(db.Cond{"grid_id": gridID, "status": CompletedStatus, "side": side}).OrderBy(orderBy).All(&orders)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
func (s *Storage) GetOrdersByGridID(gridID int64) ([]Orders, error) {
	var orders []Orders
	err := s.db.Collection("orders").
		Find(db.Cond{"grid_id": gridID}).
		All(&orders)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *Storage) FindOrderByOpenOrderID(openOrderID string) (*Orders, error) {
	var order Orders
	err := s.db.Collection("orders").
		Find(db.Cond{"order_id": openOrderID}).
		One(&order)
	if err != nil {
		return nil, err
	}
	return &order, nil
}
