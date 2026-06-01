package install

import (
	"net"
	"time"

	"github.com/abxda/bdp-meta-launcher/internal/platform"
)

// Los servicios del laboratorio escuchan en estos puertos (los mismos en
// Portable y en Vagrant, porque la box reproduce el stack del portable). Si
// están ocupados, ALGUNA solución ya está corriendo — y como ambas usan los
// mismos puertos, no pueden convivir.
var labPorts = []int{9870, 9200, 8888}

// OtherRunning reporta si parece haber un laboratorio (la otra solución, o
// esta misma) ya en marcha, detectado porque los puertos del stack responden.
// No distingue cuál; solo que "algo del lab está usando los puertos".
func LabRunning() bool {
	for _, p := range labPorts {
		if portOpen(p) {
			return true
		}
	}
	return false
}

// ConflictWith devuelve true si lanzar la solución `s` chocaría con un
// laboratorio ya corriendo. Como ambas soluciones comparten puertos, cualquier
// lab activo es un conflicto para lanzar otra.
func ConflictWith(s platform.Solution) bool {
	return LabRunning()
}

func portOpen(port int) bool {
	addr := net.JoinHostPort("127.0.0.1", itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 400*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [6]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
