// Copyright (c) 2022 ActiveChooN
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package filter

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	Id       int64
	Username string `filter:"searchable;filterable"`
	FullName string `filter:"param:full_name;searchable"`
	Email    string `filter:"filterable"`
	// This field is not filtered.
	Password string
}

type TestSuite struct {
	suite.Suite
	db   *gorm.DB
	mock sqlmock.Sqlmock
}

func (s *TestSuite) SetupTest() {
	var (
		db  *sql.DB
		err error
	)

	db, s.mock, err = sqlmock.New()
	s.NoError(err)
	s.NotNil(db)
	s.NotNil(s.mock)

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	})

	s.db, err = gorm.Open(dialector, &gorm.Config{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), s.db)
}

func (s *TestSuite) TearDownTest() {
	db, err := s.db.DB()
	require.NoError(s.T(), err)
	db.Close()
}

// TestFiltersBasic is a test suite for basic filters functionality.
func (s *TestSuite) TestFiltersBasic() {
	var users []User
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		URL: &url.URL{
			RawQuery: "username=sampleUser",
		},
	}

	s.mock.ExpectQuery(`^SELECT \* FROM "users" WHERE "username" = \$1`).
		WithArgs("sampleUser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "Username", "FullName", "Email", "Password"}))
	err := s.db.Model(&User{}).Scopes(FilterByQuery(ctx, FILTER)).Find(&users).Error
	s.NoError(err)
}

// Filtering for a field that is not filtered should not be performed
func (s *TestSuite) TestFiltersNotFilterable() {
	var users []User
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		URL: &url.URL{
			RawQuery: "password=samplePassword",
		},
	}
	s.mock.ExpectQuery(`^SELECT \* FROM "users" ORDER`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "Username", "FullName", "Email", "Password"}))
	err := s.db.Model(&User{}).Scopes(FilterByQuery(ctx, FILTER|ORDER_BY)).Find(&users).Error
	s.NoError(err)
}

// Filtering would not be applied if no config is provided.
func (s *TestSuite) TestFiltersNoFilterConfig() {
	var users []User
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		URL: &url.URL{
			RawQuery: "username=sampleUser",
		},
	}

	s.mock.ExpectQuery(`^SELECT \* FROM "users"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "Username", "FullName", "Email", "Password"}))
	err := s.db.Model(&User{}).Scopes(FilterByQuery(ctx, 0)).Find(&users).Error
	s.NoError(err)
}

/* // search function is disabled for now
// TestFiltersSearchable is a test suite for searchable filters functionality.
func (s *TestSuite) TestFiltersSearchable() {
	var users []User
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		URL: &url.URL{
			RawQuery: "search=John",
		},
	}

	s.mock.ExpectQuery(`^SELECT \* FROM "users" WHERE \("Username" LIKE \$1 OR "FullName" LIKE \$2\)`).
		WithArgs("%John%", "%John%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "Username", "FullName", "Email", "Password"}))
	err := s.db.Model(&User{}).Scopes(FilterByQuery(ctx, ALL)).Find(&users).Error
	s.NoError(err)
}*/

// TestFiltersPaginateOnly is a test suite for pagination functionality.
func (s *TestSuite) TestFiltersPaginateOnly() {
	var users []User
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		URL: &url.URL{
			RawQuery: "page=2&limit=10",
		},
	}

	s.mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"count"}))
	s.mock.ExpectQuery(`^SELECT \* FROM "users" ORDER BY "users"\."created_at" DESC LIMIT 10 OFFSET 10$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "Username", "FullName", "Email", "Password"}))
	err := s.db.Model(&User{}).Scopes(FilterByQuery(ctx, ALL)).Find(&users).Error
	s.NoError(err)
}

// TestFiltersOrderBy is a test suite for order by functionality.
func (s *TestSuite) TestFiltersOrderBy() {
	var users []User
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		URL: &url.URL{
			RawQuery: "order_by=Email&order_direction=asc",
		},
	}

	s.mock.ExpectQuery(`^SELECT \* FROM "users" ORDER BY "users"\."Email"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "Username", "FullName", "Email", "Password"}))
	err := s.db.Model(&User{}).Scopes(FilterByQuery(ctx, ORDER_BY)).Find(&users).Error
	s.NoError(err)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
