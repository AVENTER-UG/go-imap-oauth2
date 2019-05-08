package main

import (
	"encoding/json"
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
	User           UserInfo
)

type UserInfo struct {
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	ConnectorID string `json:"connector_id"`
}

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

	User.UserName = username

	return true
}

func init() {

	IMAPServer = os.Getenv("IMAPSERVER")
	IMAPPort = os.Getenv("IMAPPORT")
	IMAPDomain = os.Getenv("IMAPDOMAIN")
	ClientDomain = os.Getenv("CLIENTDOMAIN")
	ClientID = os.Getenv("CLIENTID")
	ClientSecret = os.Getenv("CLIENTSECRET")

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
	http.HandleFunc("/userinfo", userInfoHandler)

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

		uid := us.Get("UserID")

		return
	}
	outputHTML(w, r, "static/auth.html")
}

func userInfoHandler(w http.ResponseWriter, r *http.Request) {
	us, err := globalSessions.SessionStart(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("userInfoHandler: HTTP Error")
		return
	}

	var sesUser UserInfo

	sesUser.UserID = us.Get("UserID").(string)
	sesUser.EMail = us.Get("EMail").(string)
	sesUser.Name = us.Get("UserName").(string)
	sesUser.Sub = us.Get("UserID").(string)

	//var info []byte
	info, err := json.Marshal(User)

	if err != nil {
		log.Println("userInfoHandler: Error Create JSON")
		return
	}
	log.Println(User)

	sendJSON(info, w)
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

func sendJSON(js []byte, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(js)
}
