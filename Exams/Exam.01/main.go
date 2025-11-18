package main

import (
	"bufio"
	"container/heap"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Edge represents a connection between two cities.
type Edge struct {
	To       string
	Distance float64
}

// NodeDist is a node in the priority queue for Dijkstra.
type NodeDist struct {
	ID   string
	Dist float64
}

// PriorityQueue is a min-heap by Dist.
type PriorityQueue []NodeDist

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Dist < pq[j].Dist
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(NodeDist))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[:n-1]
	return x
}

// PlaceOption is used to populate HTML <select> elements.
type PlaceOption struct {
	ID   string
	Name string
}

// PathStep describes one hop in the final path.
type PathStep struct {
	From     string
	To       string
	Distance float64
}

// PathResult is the full shortest path and its total distance.
type PathResult struct {
	Steps []PathStep
	Total float64
}

// TemplateData is passed to the HTML template.
type TemplateData struct {
	Places        []PlaceOption
	SelectedStart string
	SelectedEnd   string
	Error         string
	Result        *PathResult
}

// Global state: city names, adjacency list, options for the form, template.
var (
	cityNames    map[string]string
	adj          map[string][]Edge
	options      []PlaceOption
	pageTemplate *template.Template
)

const pageHTML = `
<!DOCTYPE html>
<html lang="bg">
<head>
<meta charset="UTF-8">
<title>Shortest Path</title>
<style>
body { font-family: sans-serif; margin: 2rem; }
form { margin-bottom: 2rem; }
label { display: inline-block; width: 120px; }
select { min-width: 200px; }
.error { color: red; }
table { border-collapse: collapse; margin-top: 1rem; }
th, td { border: 1px solid #ccc; padding: 0.4rem 0.6rem; }
</style>
</head>
<body>
<h1>Shortest path between two cities</h1>
{{if .Error}}<p class="error">{{.Error}}</p>{{end}}
<form method="POST" action="/">
<p>
<label for="start">Start city:</label>
<select name="start" id="start">
<option value="">-- choose --</option>
{{range .Places}}
<option value="{{.ID}}" {{if eq $.SelectedStart .ID}}selected{{end}}>{{.Name}}</option>
{{end}}
</select>
</p>
<p>
<label for="end">End city:</label>
<select name="end" id="end">
<option value="">-- choose --</option>
{{range .Places}}
<option value="{{.ID}}" {{if eq $.SelectedEnd .ID}}selected{{end}}>{{.Name}}</option>
{{end}}
</select>
</p>
<p>
<button type="submit">Find path</button>
</p>
</form>
{{if .Result}}
<h2>Result</h2>
<p>Summary path: {{printf "%.2f" .Result.Total}} km</p>
<table>
<thead>
<tr><th>From</th><th>To</th><th>Distance (km)</th></tr>
</thead>
<tbody>
{{range .Result.Steps}}
<tr>
<td>{{.From}}</td>
<td>{{.To}}</td>
<td>{{printf "%.2f" .Distance}}</td>
</tr>
{{end}}
</tbody>
</table>
{{end}}
</body>
</html>
`

// loadGraph reads the map file and builds cityNames and adj.
func loadGraph(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	mode := "places" // "places" or "edges"
	cityNames = make(map[string]string)
	adj = make(map[string][]Edge)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if mode == "places" {
				mode = "edges"
			}
			continue
		}

		if mode == "places" {
			parts := strings.SplitN(line, ",", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid place line: %q", line)
			}
			id := strings.TrimSpace(parts[0])
			name := strings.TrimSpace(parts[1])
			if id == "" || name == "" {
				return fmt.Errorf("invalid place line: %q", line)
			}
			cityNames[id] = name
		} else {
			parts := strings.Split(line, ",")
			if len(parts) < 3 {
				return fmt.Errorf("invalid edge line: %q", line)
			}
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			id1 := parts[0]
			id2 := parts[1]

			// Join remaining tokens to support "12,34" → "12.34".
			distStr := strings.Join(parts[2:], ".")
			d, err := strconv.ParseFloat(distStr, 64)
			if err != nil {
				return fmt.Errorf("invalid distance in line %q: %v", line, err)
			}

			if _, ok := cityNames[id1]; !ok {
				return fmt.Errorf("unknown city id %q in edges", id1)
			}
			if _, ok := cityNames[id2]; !ok {
				return fmt.Errorf("unknown city id %q in edges", id2)
			}

			adj[id1] = append(adj[id1], Edge{To: id2, Distance: d})
			adj[id2] = append(adj[id2], Edge{To: id1, Distance: d})
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	options = make([]PlaceOption, 0, len(cityNames))
	for id, name := range cityNames {
		options = append(options, PlaceOption{ID: id, Name: name})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Name < options[j].Name
	})
	return nil
}

// edgeDistance returns the distance between two directly connected cities.
func edgeDistance(from, to string) float64 {
	for _, e := range adj[from] {
		if e.To == to {
			return e.Distance
		}
	}
	return 0
}

// shortestPath finds the shortest path between start and end using Dijkstra.
func shortestPath(start, end string) (*PathResult, bool) {
	dist := make(map[string]float64)
	prev := make(map[string]string)

	dist[start] = 0

	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, NodeDist{ID: start, Dist: 0})

	for pq.Len() > 0 {
		nd := heap.Pop(pq).(NodeDist)

		if nd.Dist > dist[nd.ID] {
			continue
		}

		if nd.ID == end {
			break
		}

		for _, e := range adj[nd.ID] {
			ndist := nd.Dist + e.Distance
			if d, ok := dist[e.To]; !ok || ndist < d {
				dist[e.To] = ndist
				prev[e.To] = nd.ID
				heap.Push(pq, NodeDist{ID: e.To, Dist: ndist})
			}
		}
	}

	total, ok := dist[end]
	if !ok {
		return nil, false
	}

	pathIDs := []string{}
	at := end
	for {
		pathIDs = append(pathIDs, at)
		if at == start {
			break
		}
		p, ok := prev[at]
		if !ok {
			break
		}
		at = p
	}
	if pathIDs[len(pathIDs)-1] != start {
		return nil, false
	}

	for i, j := 0, len(pathIDs)-1; i < j; i, j = i+1, j-1 {
		pathIDs[i], pathIDs[j] = pathIDs[j], pathIDs[i]
	}

	steps := make([]PathStep, 0, len(pathIDs)-1)
	for i := 0; i+1 < len(pathIDs); i++ {
		from := pathIDs[i]
		to := pathIDs[i+1]
		w := edgeDistance(from, to)
		steps = append(steps, PathStep{
			From:     cityNames[from],
			To:       cityNames[to],
			Distance: w,
		})
	}

	return &PathResult{
		Steps: steps,
		Total: total,
	}, true
}

// handler renders the form and handles shortest path requests.
func handler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Places: options,
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		data.Error = "Invalid request"
		_ = pageTemplate.Execute(w, data)
		return
	}

	if r.Method == http.MethodPost {
		start := r.FormValue("start")
		end := r.FormValue("end")

		data.SelectedStart = start
		data.SelectedEnd = end

		if start == "" || end == "" {
			data.Error = "Please choose start and end city."
		} else if _, ok := cityNames[start]; !ok {
			data.Error = "Invalid start city."
		} else if _, ok := cityNames[end]; !ok {
			data.Error = "Invalid end city."
		} else {
			res, ok := shortestPath(start, end)
			if !ok {
				data.Error = "There is no starting path bewteen the chosen cities."
			} else {
				data.Result = res
			}
		}
	}

	if err := pageTemplate.Execute(w, data); err != nil {
		log.Println("template execute:", err)
	}
}

// main parses flags, loads the graph, and starts the HTTP server.
func main() {
	fileFlag := flag.String("file", "", "path to map file")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	filename := *fileFlag
	if filename == "" {
		if len(flag.Args()) > 0 {
			filename = flag.Args()[0]
		} else {
			log.Fatal("File path must be added.")
		}
	}

	if err := loadGraph(filename); err != nil {
		log.Fatal(err)
	}

	tmpl, err := template.New("page").Parse(pageHTML)
	if err != nil {
		log.Fatal(err)
	}
	pageTemplate = tmpl

	http.HandleFunc("/", handler)

	log.Println("Server listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
