;; Here's a useful example; automatically gofmt Go buffers.
;; Watch out though, as this function wipes the undos.
(emacsdefinecmd "go-fmt" filterbuffer "gofmt")
(bindkeymode "go" "M-m g f" "go-fmt")
(addsavehook "go" "go-fmt")
