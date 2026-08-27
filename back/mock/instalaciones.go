package mock

import (
	"fmt"
	"math/rand"
	"time"

	"back/domain"
)

type InstalacionStore struct {
	store *MockStore[domain.Instalacion]
}

func NewInstalacionStore() *InstalacionStore {
	s := &InstalacionStore{store: NewMockStore[domain.Instalacion](nil)}
	s.seed()
	return s
}

var instZones = []string{"CENTRO", "NORTE", "SUR", "ESTE", "OESTE"}

var instZoneCenter = map[string][2]float64{
	"CENTRO": {-11.9686, -77.0738},
	"NORTE":  {-11.962, -77.071},
	"SUR":    {-11.975, -77.074},
	"ESTE":   {-11.969, -77.065},
	"OESTE":  {-11.967, -77.083},
}

var instStreets = map[string][]string{
	"CENTRO": {"Av. Carlos Izaguirre", "Av. Universitaria", "Jr. Los Alisos", "Av. Santa Elvira", "Jr. Marañón"},
	"NORTE":  {"Av. Antúnez de Mayolo", "Av. El Sol", "Av. Los Precursores", "Jr. Las Palmeras", "Av. Señor de los Milagros"},
	"SUR":    {"Av. Alfredo Mendiola", "Av. Universitaria", "Av. Los Alisos", "Jr. Las Orquídeas", "Av. Canta Callao"},
	"ESTE":   {"Av. Tomás Valle", "Av. Marañón", "Av. Izaguirre", "Jr. Los Geranios", "Av. Sta. Ana"},
	"OESTE":  {"Av. Elmer Faucett", "Av. Néstor Gambeta", "Av. Universitaria", "Av. Argentina", "Av. Provincias Unidas"},
}

var instFabricantes = []string{"Hikvision", "Dahua", "Samsung", "LG", "DELTA", "Sony"}
var instModelos = []string{"DS-2CD", "IPC-HFW", "QM950", "VX55", "LED-75P", "SN240"}
var instProveedores = []string{"Andina Comunicaciones", "Redes Perú", "Soluciones del Norte", "TechSur", "Integradora Lima"}
var instInstaladores = []string{"Equipo A", "Equipo B", "Equipo C", "Equipo D"}
var instContactos = []string{"Ing. Rojas", "Sr. Quispe", "Lic. Mendoza", "Téc. Paredes"}
var instContenidos = []string{
	"Campaña de seguridad vial",
	"Información municipal",
	"Publicidad local",
	"Aviso preventivo",
	"Bienvenida a la zona",
	"Estado de emergencia",
}
var instDescripciones = []string{
	"Instalación urbana de monitoreo con respaldo.",
	"Punto registrado en plan de cobertura de la zona.",
	"Equipo con soporte 24/7 y respaldo energético.",
	"Unidad asignada al contrato de mantenimiento anual.",
}

func instPick(rng *rand.Rand, pool []string) string {
	return pool[rng.Intn(len(pool))]
}

func instPickEstado(rng *rand.Rand) string {
	r := rng.Float64()
	switch {
	case r < 0.34:
		return "ACTIVO"
	case r < 0.48:
		return "INACTIVO"
	case r < 0.62:
		return "MANTENIMIENTO"
	case r < 0.74:
		return "MALOGRADO"
	case r < 0.87:
		return "PENDIENTE_INSTALACION"
	default:
		return "PENDIENTE_REPARACION"
	}
}

func instNombre(tipo, zona string, i int) string {
	n := fmt.Sprintf("%03d", i+1)
	z := zona[0:1] + toLowerRest(zona)
	switch tipo {
	case "CAMARA":
		return "Cámara " + z + " " + n
	case "PANEL":
		return "Panel LED " + z + " " + n
	case "BANNER":
		return "Banner " + z + " " + n
	case "SENSOR":
		return "Sensor " + z + " " + n
	default:
		return "Router " + z + " " + n
	}
}

func toLowerRest(s string) string {
	b := []byte(s)
	for i := 1; i < len(b); i++ {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func instRandomIP(rng *rand.Rand) string {
	return fmt.Sprintf("10.%d.%d.%d", 20+rng.Intn(10), rng.Intn(255), 2+rng.Intn(250))
}

func instRandomMAC(rng *rand.Rand) string {
	hex := func() string {
		return fmt.Sprintf("%02X", rng.Intn(256))
	}
	return hex() + ":" + hex() + ":" + hex() + ":" + hex() + ":" + hex() + ":" + hex()
}

func instRandomFirmware(rng *rand.Rand) string {
	return fmt.Sprintf("v%.1f.%d", 1+rng.Float64()*5, rng.Intn(10))
}

func instFloat(v float64) *float64 { return &v }
func instStr(v string) *string     { return &v }

func (s *InstalacionStore) seed() {
	rng := rand.New(rand.NewSource(20240827))
	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	span := end.Sub(start)

	tipos := []string{"CAMARA", "CAMARA", "PANEL", "BANNER", "SENSOR", "ROUTER"}

	for i := 0; i < 50; i++ {
		id := int64(i + 1)
		zona := instZones[i%len(instZones)]
		center := instZoneCenter[zona]
		tipo := instPick(rng, tipos)
		estado := instPickEstado(rng)

		lat := round5(center[0] + (rng.Float64()-0.5)*0.006)
		lng := round5(center[1] + (rng.Float64()-0.5)*0.008)

		fecha := start.Add(time.Duration(rng.Float64() * float64(span))).Format("2006-01-02")

		item := domain.Instalacion{
			ID:               id,
			Tipo:             tipo,
			Nombre:           instNombre(tipo, zona, i),
			Estado:           estado,
			Lat:              lat,
			Lng:              lng,
			Direccion:        fmt.Sprintf("%s %d", instPick(rng, instStreets[zona]), 100+rng.Intn(800)),
			Zona:             zona,
			FechaInstalacion: fecha,
			Serial:           fmt.Sprintf("SN-%05d", id),
			CreadoEn:         time.Now().Add(-time.Duration(rng.Intn(400)) * 24 * time.Hour),
			ActualizadoEn:    time.Now().Add(-time.Duration(rng.Intn(60)) * 24 * time.Hour),
		}

		switch estado {
		case "ACTIVO", "MANTENIMIENTO":
			uc := time.Now().Add(-time.Duration(rng.Float64()*10) * 24 * time.Hour).Format(time.RFC3339)
			item.UltimaConexion = &uc
		}

		if rng.Float64() > 0.35 {
			item.Fabricante = instStr(instPick(rng, instFabricantes))
		}
		if rng.Float64() > 0.45 {
			item.Modelo = instStr(instPick(rng, instModelos))
		}
		if rng.Float64() > 0.4 {
			item.Instalador = instStr(instPick(rng, instInstaladores))
		}
		if rng.Float64() > 0.35 {
			item.Proveedor = instStr(instPick(rng, instProveedores))
		}
		if rng.Float64() > 0.4 {
			item.Contacto = instStr(instPick(rng, instContactos))
		}
		if rng.Float64() > 0.5 {
			item.Descripcion = instStr(instPick(rng, instDescripciones))
		}
		if rng.Float64() > 0.45 {
			item.Firmware = instStr(instRandomFirmware(rng))
		}
		if estado == "ACTIVO" || estado == "MANTENIMIENTO" {
			if rng.Float64() > 0.4 {
				senal := mathRound1(40 + rng.Float64()*58)
				item.Senal = &senal
			}
		}

		switch tipo {
		case "CAMARA":
			item.Resolucion = instStr(instPick(rng, []string{"1080p", "2K", "4K", "5MP"}))
			item.IP = instStr(instRandomIP(rng))
			item.MAC = instStr(instRandomMAC(rng))
		case "PANEL":
			item.Contenido = instStr(instPick(rng, instContenidos))
			item.Potencia = instFloat(float64(120 + rng.Intn(2880)))
		case "BANNER":
			item.Contenido = instStr(instPick(rng, instContenidos))
			item.Potencia = instFloat(float64(60 + rng.Intn(900)))
		case "SENSOR":
			bateria := float64(rng.Intn(101))
			item.Bateria = &bateria
			if item.Senal == nil {
				senal := mathRound1(30 + rng.Float64()*70)
				item.Senal = &senal
			}
		case "ROUTER":
			item.IP = instStr(instRandomIP(rng))
			item.MAC = instStr(instRandomMAC(rng))
		}

		s.store.Insert(id, item)
	}
}

func round5(v float64) float64 {
	return mathRound(v*1e5) / 1e5
}

func mathRound(v float64) float64 {
	i := int64(v)
	if v < 0 {
		if v-float64(i) <= -0.5 {
			return float64(i - 1)
		}
		return float64(i)
	}
	if v-float64(i) >= 0.5 {
		return float64(i + 1)
	}
	return float64(i)
}

func mathRound1(v float64) float64 {
	return mathRound(v*10) / 10
}

// List devuelve el listado completo, ordenado por ID (una sola llamada trae todo).
func (s *InstalacionStore) List() []domain.Instalacion {
	all := s.store.FindAll()
	items := make([]domain.Instalacion, 0, len(all))
	for _, i := range all {
		items = append(items, i)
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].ID < items[j-1].ID; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	return items
}

func (s *InstalacionStore) FindByID(id int64) (*domain.Instalacion, error) {
	i, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (s *InstalacionStore) Create(i domain.Instalacion) (domain.Instalacion, error) {
	now := time.Now()
	i.CreadoEn = now
	i.ActualizadoEn = now
	return s.store.Create(i)
}

func (s *InstalacionStore) Update(id int64, i domain.Instalacion) error {
	i.ActualizadoEn = time.Now()
	return s.store.Update(id, i)
}

func (s *InstalacionStore) Delete(id int64) error {
	return s.store.Delete(id)
}
