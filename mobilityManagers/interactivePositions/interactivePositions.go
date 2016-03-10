package interactivePositions

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"path"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type interactivePositions struct {
	positionManager squirrel.PositionManager
	newPositions    chan *squirrel.Position
	laddr           string
}

func NewInteractivePositions() squirrel.MobilityManager {
	return &interactivePositions{newPositions: make(chan *squirrel.Position)}
}

func (m *interactivePositions) ParametersHelp() string {
	return ``
}

func (m *interactivePositions) Configure(conf *etcd.Node) error {
	if conf == nil {
		return errors.New("InteractivePositions: conf (*etcd.Node) is nil")
	}

	found := false
	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "/laddr") {
			m.laddr = node.Value
			found = true
			break
		}
	}
	if !found {
		return errors.New("laddr is missing from config")
	}
	return nil
}

func (m *interactivePositions) Initialize(positionManager squirrel.PositionManager) {
	m.positionManager = positionManager
	go http.ListenAndServe(m.laddr, m.bindMux())
}

type JSPosition struct {
	I int
	X float64
	Y float64
	H float64
}

func positionFromPosition(i int, p *squirrel.Position) *JSPosition {
	return &JSPosition{I: i, X: p.X, Y: p.Y, H: p.Height}
}

func (m *interactivePositions) bindMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/list", func(w http.ResponseWriter, req *http.Request) {
		ret := make([]*JSPosition, 0)
		for _, index := range m.positionManager.Enabled() {
			p, err := m.positionManager.Get(index)
			if err == nil {
				ret = append(ret, positionFromPosition(index, &p))
			}
		}
		json.NewEncoder(w).Encode(ret)
	})
	mux.HandleFunc("/set", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			http.NotFound(w, req)
			return
		}
		var pos JSPosition
		err := json.NewDecoder(req.Body).Decode(&pos)
		if nil != err {
			http.Error(w, "json Decoding error", 500)
			return
		}
		m.positionManager.Set(pos.I, pos.X, pos.Y, pos.H)
	})
	pkgRoot, err := getRootPath()
	if err != nil {
		return nil
	}
	mux.Handle("/", http.FileServer(http.Dir(path.Join(pkgRoot, "assets"))))

	return mux
}

func getRootPath() (string, error) {
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", "github.com/squirrel-land/models/mobilityManagers/interactivePositions").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
