package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	"github.com/wxdao/go-imap/imap"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
	"gopkg.in/session.v1"
)

var (
	globalSessions *session.Manager
	IMAPServer     string
	IMAPDomain     string
	IMAPPort       string
	ClientDomain   string
	ClientID       string
	ClientSecret   string
)

func ImapLogin(username, userPassword string) bool {
	log.Println("authenticate against imap", username)
	client, err := imap.Dial(IMAPServer + ":" + IMAPPort)
	if err != nil {
		panic(err)
	}

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, os.Kill)

	updated := make(chan int)

	// invoked when status changed
	client.UpdateCallback = func() {
		updated <- 1
	}

	client.StartTLS(IMAPServer)
	err = client.Login(username+"@"+IMAPDomain, userPassword)
	if err != nil {
		log.Println("imap login was not successfull")
		return false
	}
	log.Println("imap login success")
	return true
}

func init() {

	flag.StringVar(&IMAPServer, "imapserver", "mail.gmail.com", "imap server")
	flag.StringVar(&IMAPPort, "imapport", "143", "imap port")
	flag.StringVar(&IMAPDomain, "imapdomain", "gmail.com", "imab domain")
	flag.StringVar(&ClientDomain, "clientdomain", "http://localhost", "client domain")
	flag.StringVar(&ClientID, "clientid", "222222", "client id")
	flag.StringVar(&ClientSecret, "clientsecret", "22222222", "client secret")

	flag.Parse()

	fmt.Println("IMAPServer=", IMAPServer)
	fmt.Println("IMAPPort=", IMAPPort)
	fmt.Println("IMAPDomain=", IMAPDomain)
	fmt.Println("ClientDomain=", ClientDomain)
	fmt.Println("ClientSecret=", ClientSecret)
	fmt.Println("ClientID=", ClientID)

	globalSessions, _ = session.NewManager("memory", `{"cookieName":"gosessionid","gclifetime":3600}`)
	go globalSessions.GC()
}

func main() {
	manager := manage.NewDefaultManager()
	// token store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	clientStore := store.NewClientStore()
	clientStore.Set(ClientID, &models.Client{
		ID:     ClientID,
		Secret: ClientSecret,
		Domain: ClientDomain,
	})
	manager.MapClientStorage(clientStore)

	srv := server.NewServer(server.NewConfig(), manager)
	srv.SetUserAuthorizationHandler(userAuthorizeHandler)

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/auth", authHandler)

	http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleTokenRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Println("Server is running at 9096 port.")
	log.Fatal(http.ListenAndServe(":9096", nil))
}

func userAuthorizeHandler(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	us, err := globalSessions.SessionStart(w, r)
	uid := us.Get("UserID")

	if uid == nil {
		if r.Form == nil {
			r.ParseForm()
		}
		us.Set("Form", r.Form)
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
		return
	}
	userID = uid.(string)
	us.Delete("UserID")
	return
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		us, err := globalSessions.SessionStart(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username := r.FormValue("username")
		userPassword := r.FormValue("password")

		if ImapLogin(username, userPassword) == true {
			us.Set("LoggedInUserID", username)
			w.Header().Set("Location", "/auth")
			w.WriteHeader(http.StatusFound)
			return
		}
	}
	outputHTML(w, r, "static/login.html")
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	us, err := globalSessions.SessionStart(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if us.Get("LoggedInUserID") == nil {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
		return
	}
	if r.Method == "POST" {

		form := us.Get("Form").(url.Values)
		u := new(url.URL)
		u.Path = "/authorize"
		u.RawQuery = form.Encode()
		w.Header().Set("Location", u.String())
		w.WriteHeader(http.StatusFound)
		us.Delete("Form")
		us.Set("UserID", us.Get("LoggedInUserID"))
		return
	}
	outputHTML(w, r, "static/auth.html")
}

func outputHTML(w http.ResponseWriter, req *http.Request, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()
	fi, _ := file.Stat()
	http.ServeContent(w, req, file.Name(), fi.ModTime(), file)
}
