package cli

// Типы секретов — совпадают с константами model.Secret на сервере.
const (
	typeLogin  = "login"
	typeText   = "text"
	typeBinary = "binary"
	typeCard   = "card"
)

// loginPayload — расшифрованный контент секрета типа "login".
type loginPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// textPayload — контент секрета типа "text".
type textPayload struct {
	Content string `json:"content"`
}

// cardPayload — контент секрета типа "card".
type cardPayload struct {
	Number  string `json:"number"`
	Expires string `json:"expires"`
	Holder  string `json:"holder"`
	CVV     string `json:"cvv"`
}
