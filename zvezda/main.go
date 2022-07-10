package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/muesli/coral"
	log "github.com/sirupsen/logrus"
	"github.com/stianeikeland/go-rpio/v4"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/gorm"
)

var (
	rootCmd = &coral.Command{
		Use: "zvezda",
	}

	InnerdoorPin rpio.Pin
	OuterdoorPin rpio.Pin
)

type CtxKey string

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"method": r.Method,
			"url":    r.URL,
		}).Info("received request")
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(db *gorm.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expecting basic auth
		username, password, ok := r.BasicAuth()
		if !ok {
			log.Info("No authentication credentials")
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var user User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			log.WithField("username", username).Error(err)
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Hash), []byte(password)); err != nil {
			log.WithField("username", username).Error(err)
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		user.LastUsedAt = time.Now()
		db.Save(&user)

		ctx := context.WithValue(r.Context(), CtxKey("username"), username)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})

}

var serveCmd = &coral.Command{
	Use: "serve",
	RunE: func(_ *coral.Command, _ []string) error {
		pinIdInnerdoors, err := strconv.Atoi(os.Getenv("PIN_INNERDOOR"))
		if err != nil {
			return err
		}
		pinIdOuterdoors, err := strconv.Atoi(os.Getenv("PIN_OUTERDOOR"))
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"pinIdInnerdoors": pinIdInnerdoors,
			"pinIdOuterdoors": pinIdOuterdoors,
		}).Info("got environment variables")

		log.Info("Opening GPIO")
		err = rpio.Open()
		if err != nil {
			return err
		}
		InnerdoorPin = rpio.Pin(pinIdInnerdoors)
		InnerdoorPin.Output()
		OuterdoorPin = rpio.Pin(pinIdOuterdoors)
		OuterdoorPin.Output()

		db, err := getDatabase()
		if err != nil {
			return err
		}

		mux := http.NewServeMux()
		innentuereLastOpened := time.Now().Add(-10 * time.Second)
		aussentuereLastOpened := time.Now().Add(-10 * time.Second)

		mux.HandleFunc("/open/innerdoor", func(w http.ResponseWriter, r *http.Request) {
			if innentuereLastOpened.Add(10 * time.Second).After(time.Now()) {
				w.WriteHeader(http.StatusTooEarly)
				return
			}
			innentuereLastOpened = time.Now()
			log.WithField("username", r.Context().Value(CtxKey("username"))).Info("Innerdoor opened")
			log.WithField("pin", InnerdoorPin).Info("pin high")
			InnerdoorPin.High()
			time.Sleep(5 * time.Second)
			log.WithField("pin", InnerdoorPin).Info("pin low")
			InnerdoorPin.Low()
		})

		mux.HandleFunc("/open/outerdoor", func(w http.ResponseWriter, r *http.Request) {
			if aussentuereLastOpened.Add(10 * time.Second).After(time.Now()) {
				w.WriteHeader(http.StatusTooEarly)
				return
			}
			aussentuereLastOpened = time.Now()
			log.WithField("username", r.Context().Value(CtxKey("username"))).Info("Outerdoor opened")
			log.WithField("pin", OuterdoorPin).Info("pin high")
			OuterdoorPin.High()
			time.Sleep(5 * time.Second)
			log.WithField("pin", OuterdoorPin).Info("pin low")
			OuterdoorPin.Low()
		})

		fs, err := getStaticFS()
		if err != nil {
			return err
		}
		mux.Handle("/", http.FileServer(http.FS(fs)))

		log.WithField("port", "3000").Info("ListenAndServe")
		return http.ListenAndServe(":3000", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loggingMiddleware(authMiddleware(db, mux)).ServeHTTP(w, r)
		}))
	},
}

var userCmd = &coral.Command{
	Use: "user",
}

var userAddCmd = &coral.Command{
	Use: "add <username>",
	RunE: func(cmd *coral.Command, args []string) error {
		db, err := getDatabase()
		if err != nil {
			return err
		}
		log.Info(db)
		if len(args) < 1 {
			return fmt.Errorf("which user do you want to add?")
		}
		if len(args) > 1 {
			return fmt.Errorf("please only enter a single user name")
		}
		username := args[0]
		fmt.Printf("Password for %s: ", username)
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Printf("\n")
		if len(passwordBytes) < 8 {
			return fmt.Errorf("password must be at least 8 characters long")
		}

		hash, err := bcrypt.GenerateFromPassword(passwordBytes, -1)
		if err != nil {
			return err
		}

		if err := db.Create(&User{
			Username: username,
			Hash:     hash,
		}).Error; err != nil {
			return err
		}
		return nil
	},
}

var userUpdateCmd = &coral.Command{
	Use: "update <user>",
	RunE: func(cmd *coral.Command, args []string) error {
		db, err := getDatabase()
		if err != nil {
			return err
		}
		log.Info(db)

		if len(args) < 1 {
			return fmt.Errorf("which user do you want to add?")
		}
		if len(args) > 1 {
			return fmt.Errorf("please only enter a single user name")
		}
		username := args[0]
		fmt.Printf("Password for %s: ", username)
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Printf("\n")
		if len(passwordBytes) < 8 {
			return fmt.Errorf("password must be at least 8 characters long")
		}

		hash, err := bcrypt.GenerateFromPassword(passwordBytes, -1)
		if err != nil {
			return err
		}

		if err := db.Model(&User{
			Username: username,
		}).Updates(&User{
			Hash: hash,
		}).Error; err != nil {
			return err
		}
		log.Infof("user %s updated", username)
		return nil
	},
}

//go:embed static
var f embed.FS

func getStaticFS() (fs.FS, error) {
	if developerMode {
		log.Warn("developer mode enabled, serving local files")
		return os.DirFS("static"), nil
	}
	log.Info("developer mode disabled, serving embedded files")
	return fs.Sub(f, "static")
}

type User struct {
	Username   string `gorm:"primaryKey"`
	Hash       []byte
	LastUsedAt time.Time
	CreatedAt  time.Time
}

var db *gorm.DB

func getDatabase() (*gorm.DB, error) {
	log.Info("Connecting to DB")
	var err error
	if db == nil {
		db, err = gorm.Open(sqlite.Open("zvezda.db"))
		if err != nil {
			return nil, err
		}
	}

	db.AutoMigrate(&User{})

	return db, nil
}

var developerMode bool

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().BoolVar(&developerMode, "dev", false, "developer mode. Enables files from disk instead of files embeded into binary.")
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userUpdateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
