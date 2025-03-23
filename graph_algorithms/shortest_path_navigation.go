package graph_algorithms

/*
最短路径导航系统

原理：
最短路径导航系统基于图论中的最短路径算法，将现实世界的地理位置和道路网络抽象为图模型，
通过计算图中节点间的最短路径来规划从起点到终点的最优路线。

关键特点：
1. 基于图模型，将地点抽象为节点，道路抽象为边
2. 支持边权重（如距离、时间、交通状况等）
3. 实现多种最短路径算法（如Dijkstra、A*）
4. 可处理复杂的约束条件（如避开收费站、偏好高速公路等）
5. 支持实时路况更新和动态路径规划

实现方式：
- 使用邻接表或邻接矩阵表示图结构
- 实现优先级队列辅助最短路径算法
- 支持多种路径评估指标（距离、时间、花费等）
- 提供路径可视化和导航指令

应用场景：
- 车辆导航系统
- 物流路径规划
- 公共交通换乘指南
- 步行/骑行路线规划
- 网络路由优化

优缺点：
- 优点：提供高效的路线规划，节省时间和资源
- 缺点：算法复杂度较高，计算大规模图的最短路径需要优化

以下实现了基于Dijkstra和A*算法的最短路径导航系统。
*/

import (
	"container/heap"
	"fmt"
	"math"
)

// 位置坐标（用于A*算法的启发式函数）
type Coordinate struct {
	X, Y float64 // 坐标点的X、Y值（可以是经纬度）
}

// 计算两点间的欧几里得距离
func (c Coordinate) Distance(other Coordinate) float64 {
	dx := c.X - other.X
	dy := c.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// 图中的节点
type Node struct {
	ID          string     // 节点唯一标识
	Name        string     // 节点名称（如城市、交叉口名）
	Coordinate  Coordinate // 节点的地理坐标
	Connections []*Edge    // 从此节点出发的边
}

// 图中的边
type Edge struct {
	From     *Node   // 起始节点
	To       *Node   // 目标节点
	Weight   float64 // 边的权重（如距离、时间）
	RoadType string  // 道路类型（如高速、国道、省道）
	Toll     bool    // 是否收费
}

// 导航图
type NavigationGraph struct {
	Nodes map[string]*Node // 图中所有节点
}

// 创建新的导航图
func NewNavigationGraph() *NavigationGraph {
	return &NavigationGraph{
		Nodes: make(map[string]*Node),
	}
}

// 添加节点
func (g *NavigationGraph) AddNode(id, name string, x, y float64) *Node {
	node := &Node{
		ID:          id,
		Name:        name,
		Coordinate:  Coordinate{X: x, Y: y},
		Connections: make([]*Edge, 0),
	}
	g.Nodes[id] = node
	return node
}

// 添加边
func (g *NavigationGraph) AddEdge(fromID, toID string, weight float64, roadType string, toll bool) bool {
	fromNode, fromExists := g.Nodes[fromID]
	toNode, toExists := g.Nodes[toID]

	if !fromExists || !toExists {
		return false
	}

	// 创建并添加边
	edge := &Edge{
		From:     fromNode,
		To:       toNode,
		Weight:   weight,
		RoadType: roadType,
		Toll:     toll,
	}
	fromNode.Connections = append(fromNode.Connections, edge)
	return true
}

// 用于Dijkstra算法的优先级队列项
type DijkstraItem struct {
	NodeID   string  // 节点ID
	Distance float64 // 从起点到此节点的距离
	Index    int     // 在堆中的索引
}

// 优先级队列实现
type PathPriorityQueue []*DijkstraItem

func (pq PathPriorityQueue) Len() int { return len(pq) }

func (pq PathPriorityQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}

func (pq PathPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PathPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*DijkstraItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PathPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// 路径规划选项
type RouteOptions struct {
	AvoidTolls        bool     // 避开收费道路
	PreferredRoads    []string // 偏好的道路类型
	MaxDistance       float64  // 最大距离限制
	UseAStarAlgorithm bool     // 是否使用A*算法
}

// 路径结果
type Route struct {
	Path       []*Node  // 路径上的节点序列
	Distance   float64  // 总距离
	Tolls      int      // 收费站数量
	Directions []string // 导航指令
}

// 使用Dijkstra算法计算最短路径
func (g *NavigationGraph) FindShortestPath(fromID, toID string, options RouteOptions) (*Route, error) {
	// 验证起点和终点存在
	startNode, exists := g.Nodes[fromID]
	if !exists {
		return nil, fmt.Errorf("起点节点不存在: %s", fromID)
	}

	endNode, exists := g.Nodes[toID]
	if !exists {
		return nil, fmt.Errorf("终点节点不存在: %s", toID)
	}

	// 如果选择使用A*算法
	if options.UseAStarAlgorithm {
		return g.findShortestPathAStar(startNode, endNode, options)
	}

	// 默认使用Dijkstra算法
	return g.findShortestPathDijkstra(startNode, endNode, options)
}

// Dijkstra算法实现
func (g *NavigationGraph) findShortestPathDijkstra(startNode, endNode *Node, options RouteOptions) (*Route, error) {
	// 初始化距离表和前驱节点表
	distances := make(map[string]float64)
	previous := make(map[string]string)
	for id := range g.Nodes {
		distances[id] = math.Inf(1)
	}
	distances[startNode.ID] = 0

	// 初始化优先级队列
	pq := make(PathPriorityQueue, 0)
	heap.Init(&pq)
	heap.Push(&pq, &DijkstraItem{
		NodeID:   startNode.ID,
		Distance: 0,
	})

	// 开始Dijkstra算法
	for pq.Len() > 0 {
		// 获取当前距离最小的节点
		current := heap.Pop(&pq).(*DijkstraItem)
		currentNode := g.Nodes[current.NodeID]

		// 如果已经到达终点，可以提前结束
		if current.NodeID == endNode.ID {
			break
		}

		// 如果当前的距离已经大于已知距离，跳过
		if current.Distance > distances[current.NodeID] {
			continue
		}

		// 遍历当前节点的所有边
		for _, edge := range currentNode.Connections {
			// 检查是否符合路由选项
			if options.AvoidTolls && edge.Toll {
				continue
			}

			// 计算新的距离
			newDistance := distances[current.NodeID] + edge.Weight

			// 如果找到了更短的路径
			if newDistance < distances[edge.To.ID] {
				distances[edge.To.ID] = newDistance
				previous[edge.To.ID] = current.NodeID

				// 添加到优先级队列
				heap.Push(&pq, &DijkstraItem{
					NodeID:   edge.To.ID,
					Distance: newDistance,
				})
			}
		}
	}

	// 检查是否找到路径
	if math.IsInf(distances[endNode.ID], 1) {
		return nil, fmt.Errorf("无法找到从 %s 到 %s 的路径", startNode.Name, endNode.Name)
	}

	// 构建路径
	route := &Route{
		Path:     make([]*Node, 0),
		Distance: distances[endNode.ID],
		Tolls:    0,
	}

	// 从终点回溯到起点，构建路径
	for at := endNode.ID; at != ""; at = previous[at] {
		route.Path = append([]*Node{g.Nodes[at]}, route.Path...)
		if at == startNode.ID {
			break
		}
	}

	// 生成导航指令
	route.Directions = g.generateDirections(route.Path)

	// 计算收费站数量
	for i := 0; i < len(route.Path)-1; i++ {
		for _, edge := range route.Path[i].Connections {
			if edge.To.ID == route.Path[i+1].ID && edge.Toll {
				route.Tolls++
			}
		}
	}

	return route, nil
}

// A*算法实现
func (g *NavigationGraph) findShortestPathAStar(startNode, endNode *Node, options RouteOptions) (*Route, error) {
	// 初始化开放集、关闭集、距离表和前驱节点表
	openSet := make(map[string]bool)
	closedSet := make(map[string]bool)
	gScore := make(map[string]float64)
	fScore := make(map[string]float64)
	previous := make(map[string]string)

	// 初始化起点数据
	openSet[startNode.ID] = true
	gScore[startNode.ID] = 0
	fScore[startNode.ID] = startNode.Coordinate.Distance(endNode.Coordinate)

	// 初始化优先级队列（基于f-score）
	pq := make(PathPriorityQueue, 0)
	heap.Init(&pq)
	heap.Push(&pq, &DijkstraItem{
		NodeID:   startNode.ID,
		Distance: fScore[startNode.ID],
	})

	// 启动A*算法主循环
	for len(openSet) > 0 {
		// 获取当前f-score最小的节点
		current := heap.Pop(&pq).(*DijkstraItem)
		currentNode := g.Nodes[current.NodeID]

		// 如果已经处理过这个节点或不在开放集中，跳过
		if !openSet[current.NodeID] {
			continue
		}

		// 如果到达终点
		if current.NodeID == endNode.ID {
			// 构建路径
			route := &Route{
				Path:     make([]*Node, 0),
				Distance: gScore[endNode.ID],
				Tolls:    0,
			}

			// 从终点回溯到起点，构建路径
			for at := endNode.ID; at != ""; at = previous[at] {
				route.Path = append([]*Node{g.Nodes[at]}, route.Path...)
				if at == startNode.ID {
					break
				}
			}

			// 生成导航指令
			route.Directions = g.generateDirections(route.Path)

			// 计算收费站数量
			for i := 0; i < len(route.Path)-1; i++ {
				for _, edge := range route.Path[i].Connections {
					if edge.To.ID == route.Path[i+1].ID && edge.Toll {
						route.Tolls++
					}
				}
			}

			return route, nil
		}

		// 将当前节点从开放集移到关闭集
		delete(openSet, current.NodeID)
		closedSet[current.NodeID] = true

		// 遍历所有相邻节点
		for _, edge := range currentNode.Connections {
			neighbor := edge.To

			// 如果相邻节点在关闭集中，跳过
			if closedSet[neighbor.ID] {
				continue
			}

			// 检查是否符合路由选项
			if options.AvoidTolls && edge.Toll {
				continue
			}

			// 计算到相邻节点的临时g-score
			tentativeGScore := gScore[current.NodeID] + edge.Weight

			// 如果相邻节点不在开放集中，添加它
			if !openSet[neighbor.ID] {
				openSet[neighbor.ID] = true
			} else if tentativeGScore >= gScore[neighbor.ID] {
				// 如果这条路径不比已知路径更好，跳过
				continue
			}

			// 这是目前为止最好的路径，记录它
			previous[neighbor.ID] = current.NodeID
			gScore[neighbor.ID] = tentativeGScore
			fScore[neighbor.ID] = gScore[neighbor.ID] + neighbor.Coordinate.Distance(endNode.Coordinate)

			// 更新优先级队列
			heap.Push(&pq, &DijkstraItem{
				NodeID:   neighbor.ID,
				Distance: fScore[neighbor.ID],
			})
		}
	}

	// 如果没有找到路径
	return nil, fmt.Errorf("无法找到从 %s 到 %s 的路径", startNode.Name, endNode.Name)
}

// 生成导航指令
func (g *NavigationGraph) generateDirections(path []*Node) []string {
	if len(path) <= 1 {
		return []string{"无需导航，已在目的地"}
	}

	directions := make([]string, 0)
	directions = append(directions, fmt.Sprintf("从 %s 出发", path[0].Name))

	for i := 0; i < len(path)-1; i++ {
		current := path[i]
		next := path[i+1]

		// 查找连接这两个节点的边
		var connectingEdge *Edge
		for _, edge := range current.Connections {
			if edge.To.ID == next.ID {
				connectingEdge = edge
				break
			}
		}

		if connectingEdge != nil {
			var tollInfo string
			if connectingEdge.Toll {
				tollInfo = "（收费）"
			} else {
				tollInfo = ""
			}

			directions = append(directions, fmt.Sprintf(
				"沿 %s%s 行驶 %.1f 公里到达 %s",
				connectingEdge.RoadType,
				tollInfo,
				connectingEdge.Weight,
				next.Name,
			))
		} else {
			directions = append(directions, fmt.Sprintf("前往 %s", next.Name))
		}
	}

	directions = append(directions, fmt.Sprintf("到达目的地：%s", path[len(path)-1].Name))
	return directions
}

// 打印路径信息
func (r *Route) PrintRoute() {
	fmt.Println("\n=== 路径信息 ===")
	fmt.Printf("总距离: %.1f 公里\n", r.Distance)
	fmt.Printf("收费站数量: %d\n", r.Tolls)

	fmt.Println("\n=== 路径节点 ===")
	for i, node := range r.Path {
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Print(node.Name)
	}
	fmt.Println()

	fmt.Println("\n=== 导航指令 ===")
	for i, direction := range r.Directions {
		fmt.Printf("%d. %s\n", i+1, direction)
	}
}

// 创建示例城市地图
func createCityMap() *NavigationGraph {
	graph := NewNavigationGraph()

	// 添加城市节点
	graph.AddNode("BJ", "北京", 116.4, 39.9)
	graph.AddNode("TJ", "天津", 117.2, 39.1)
	graph.AddNode("SJZ", "石家庄", 114.5, 38.0)
	graph.AddNode("TS", "唐山", 118.2, 39.6)
	graph.AddNode("BD", "保定", 115.5, 38.9)
	graph.AddNode("CD", "承德", 117.9, 40.9)
	graph.AddNode("ZJK", "张家口", 114.9, 40.8)
	graph.AddNode("QHD", "秦皇岛", 119.6, 39.9)
	graph.AddNode("XT", "邢台", 114.5, 37.1)
	graph.AddNode("HD", "邯郸", 114.5, 36.6)

	// 添加道路连接
	graph.AddEdge("BJ", "TJ", 120, "高速公路", true)
	graph.AddEdge("TJ", "BJ", 120, "高速公路", true)
	graph.AddEdge("BJ", "SJZ", 280, "高速公路", true)
	graph.AddEdge("SJZ", "BJ", 280, "高速公路", true)
	graph.AddEdge("BJ", "BD", 140, "高速公路", true)
	graph.AddEdge("BD", "BJ", 140, "高速公路", true)
	graph.AddEdge("BJ", "ZJK", 200, "高速公路", true)
	graph.AddEdge("ZJK", "BJ", 200, "高速公路", true)
	graph.AddEdge("TJ", "TS", 170, "高速公路", true)
	graph.AddEdge("TS", "TJ", 170, "高速公路", true)
	graph.AddEdge("TJ", "QHD", 250, "高速公路", true)
	graph.AddEdge("QHD", "TJ", 250, "高速公路", true)
	graph.AddEdge("QHD", "CD", 220, "国道", false)
	graph.AddEdge("CD", "QHD", 220, "国道", false)
	graph.AddEdge("ZJK", "CD", 180, "省道", false)
	graph.AddEdge("CD", "ZJK", 180, "省道", false)
	graph.AddEdge("SJZ", "BD", 140, "高速公路", true)
	graph.AddEdge("BD", "SJZ", 140, "高速公路", true)
	graph.AddEdge("SJZ", "XT", 90, "高速公路", true)
	graph.AddEdge("XT", "SJZ", 90, "高速公路", true)
	graph.AddEdge("XT", "HD", 70, "国道", false)
	graph.AddEdge("HD", "XT", 70, "国道", false)
	graph.AddEdge("TS", "SJZ", 240, "省道", false)
	graph.AddEdge("SJZ", "TS", 240, "省道", false)

	return graph
}

// 最短路径导航示例
func ShortestPathNavigationDemo() {
	fmt.Println("== 最短路径导航系统示例 ==")

	// 创建城市地图
	cityMap := createCityMap()

	// 测试场景1：标准路径规划（北京 → 邯郸）
	fmt.Println("\n[场景1] 从北京到邯郸的标准路径规划:")
	route1, err := cityMap.FindShortestPath("BJ", "HD", RouteOptions{
		UseAStarAlgorithm: false,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		route1.PrintRoute()
	}

	// 测试场景2：避开收费路径规划（北京 → 承德）
	fmt.Println("\n[场景2] 从北京到承德的无收费路径规划:")
	route2, err := cityMap.FindShortestPath("BJ", "CD", RouteOptions{
		AvoidTolls:        true,
		UseAStarAlgorithm: false,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		route2.PrintRoute()
	}

	// 测试场景3：使用A*算法的路径规划（天津 → 石家庄）
	fmt.Println("\n[场景3] 使用A*算法从天津到石家庄的路径规划:")
	route3, err := cityMap.FindShortestPath("TJ", "SJZ", RouteOptions{
		UseAStarAlgorithm: true,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		route3.PrintRoute()
	}

	// 测试场景4：非直连城市的路径规划（秦皇岛 → 邢台）
	fmt.Println("\n[场景4] 从秦皇岛到邢台的复杂路径规划:")
	route4, err := cityMap.FindShortestPath("QHD", "XT", RouteOptions{
		UseAStarAlgorithm: true,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		route4.PrintRoute()
	}
}
