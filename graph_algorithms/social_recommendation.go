package graph_algorithms

/*
社交网络推荐系统

原理：
社交网络推荐系统利用用户的社交关系和行为数据，基于图算法分析用户之间的关联性和相似性，
从而为用户推荐可能感兴趣的内容、好友或社区。

关键特点：
1. 利用图结构表示用户之间的社交关系
2. 考虑用户之间的直接和间接连接
3. 结合用户兴趣、行为数据增强推荐效果
4. 应用各种图算法如随机游走、相似度计算等

实现方式：
- 构建用户社交图，节点为用户，边为社交关系
- 使用协同过滤、图算法分析用户间相似度
- 结合内容特征进行混合推荐
- 通过矩阵运算优化大规模计算

应用场景：
- 社交媒体的好友推荐
- 社区/兴趣小组推荐
- 内容分发和个性化信息流
- 专业社交网络中的连接推荐

优缺点：
- 优点：充分利用社交关系，提高推荐相关性
- 缺点：冷启动问题、计算复杂度较高

以下实现了基于图的社交网络推荐系统，包括好友推荐和内容推荐。
*/

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

// User 表示社交网络中的用户
type User struct {
	ID        int
	Name      string
	Interests map[string]float64 // 用户对不同兴趣的偏好程度
	Friends   map[int]bool       // 用户的直接好友（邻居节点）
}

// Post 表示社交网络中的内容
type Post struct {
	ID        int
	AuthorID  int
	Title     string
	Content   string
	Tags      []string
	Timestamp time.Time
	Likes     map[int]bool // 点赞用户ID集合
}

// SocialNetwork 表示社交网络图
type SocialNetwork struct {
	Users          map[int]*User           // 用户节点
	Posts          map[int]*Post           // 内容节点
	UserPostMatrix map[int]map[int]float64 // 用户-内容交互矩阵
}

// NewSocialNetwork 创建一个新的社交网络
func NewSocialNetwork() *SocialNetwork {
	return &SocialNetwork{
		Users:          make(map[int]*User),
		Posts:          make(map[int]*Post),
		UserPostMatrix: make(map[int]map[int]float64),
	}
}

// AddUser 添加用户到社交网络
func (sn *SocialNetwork) AddUser(user *User) {
	sn.Users[user.ID] = user
	sn.UserPostMatrix[user.ID] = make(map[int]float64)
}

// AddPost 添加内容到社交网络
func (sn *SocialNetwork) AddPost(post *Post) {
	sn.Posts[post.ID] = post
}

// AddFriendship 在两个用户之间建立好友关系
func (sn *SocialNetwork) AddFriendship(userID1, userID2 int) bool {
	user1, ok1 := sn.Users[userID1]
	user2, ok2 := sn.Users[userID2]

	if !ok1 || !ok2 {
		return false
	}

	// 添加双向好友关系
	user1.Friends[userID2] = true
	user2.Friends[userID1] = true

	return true
}

// AddInteraction 添加用户对内容的交互（例如点赞）
func (sn *SocialNetwork) AddInteraction(userID, postID int, weight float64) bool {
	_, userExists := sn.Users[userID]
	post, postExists := sn.Posts[postID]

	if !userExists || !postExists {
		return false
	}

	// 更新交互矩阵
	sn.UserPostMatrix[userID][postID] = weight

	// 如果是点赞，更新Post的点赞集合
	if weight > 0 {
		if post.Likes == nil {
			post.Likes = make(map[int]bool)
		}
		post.Likes[userID] = true
	}

	return true
}

// 计算两个用户之间的相似度（基于共同好友和共同兴趣）
func (sn *SocialNetwork) calculateUserSimilarity(userID1, userID2 int) float64 {
	user1 := sn.Users[userID1]
	user2 := sn.Users[userID2]

	// 1. 计算共同好友的Jaccard相似度
	commonFriends := 0
	for friendID := range user1.Friends {
		if user2.Friends[friendID] {
			commonFriends++
		}
	}

	totalFriends := len(user1.Friends) + len(user2.Friends) - commonFriends
	friendSimilarity := 0.0
	if totalFriends > 0 {
		friendSimilarity = float64(commonFriends) / float64(totalFriends)
	}

	// 2. 计算兴趣相似度（余弦相似度）
	interestSimilarity := 0.0
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	// 计算兴趣向量的点积和模
	for interest, weight1 := range user1.Interests {
		if weight2, ok := user2.Interests[interest]; ok {
			dotProduct += weight1 * weight2
		}
		norm1 += weight1 * weight1
	}

	for _, weight2 := range user2.Interests {
		norm2 += weight2 * weight2
	}

	// 计算余弦相似度
	if norm1 > 0 && norm2 > 0 {
		interestSimilarity = dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
	}

	// 综合计算最终相似度，这里简单地取平均值
	// 在实际应用中，可以给不同因素赋予不同权重
	return 0.6*friendSimilarity + 0.4*interestSimilarity
}

// 用于优先队列的推荐项
type RecommendationItem struct {
	ID    int     // 推荐项的ID（用户ID或内容ID）
	Score float64 // 推荐分数
	index int     // 在堆中的索引
}

// 优先队列实现
type PriorityQueue []*RecommendationItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// 分数越高的项优先级越高
	return pq[i].Score > pq[j].Score
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*RecommendationItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// RecommendFriends 为指定用户推荐好友
func (sn *SocialNetwork) RecommendFriends(userID int, count int) ([]*RecommendationItem, error) {
	user, ok := sn.Users[userID]
	if !ok {
		return nil, fmt.Errorf("用户ID %d 不存在", userID)
	}

	// 创建优先队列用于存储推荐结果
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	// 已访问过的用户集合（避免重复推荐）
	visited := make(map[int]bool)
	visited[userID] = true // 不推荐用户自己

	// 标记已经是好友的用户
	for friendID := range user.Friends {
		visited[friendID] = true
	}

	// 计算二度好友的推荐得分
	for friendID := range user.Friends {
		friend := sn.Users[friendID]

		// 遍历朋友的朋友
		for fofID := range friend.Friends {
			if !visited[fofID] {
				// 计算与这个二度好友的相似度
				similarity := sn.calculateUserSimilarity(userID, fofID)

				// 加入优先队列
				heap.Push(&pq, &RecommendationItem{
					ID:    fofID,
					Score: similarity,
				})

				visited[fofID] = true
			}
		}
	}

	// 获取前count个推荐结果
	result := make([]*RecommendationItem, 0, min(count, pq.Len()))
	for i := 0; i < count && pq.Len() > 0; i++ {
		item := heap.Pop(&pq).(*RecommendationItem)
		result = append(result, item)
	}

	return result, nil
}

// RecommendPosts 为指定用户推荐内容
func (sn *SocialNetwork) RecommendPosts(userID int, count int) ([]*RecommendationItem, error) {
	user, ok := sn.Users[userID]
	if !ok {
		return nil, fmt.Errorf("用户ID %d 不存在", userID)
	}

	// 创建优先队列用于存储推荐结果
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	// 已经交互过的内容（避免重复推荐）
	interactedPosts := make(map[int]bool)
	for postID := range sn.UserPostMatrix[userID] {
		interactedPosts[postID] = true
	}

	// 内容推荐策略：
	// 1. 好友最近发布或喜欢的内容
	// 2. 与用户兴趣相关的内容

	// 好友互动内容权重
	friendPostScores := make(map[int]float64)

	// 收集好友互动的内容
	for friendID := range user.Friends {
		// 好友创建的内容
		for postID, post := range sn.Posts {
			if post.AuthorID == friendID && !interactedPosts[postID] {
				// 根据时间新鲜度赋予权重
				age := time.Since(post.Timestamp).Hours() / 24 // 转换为天数
				timeDecay := math.Exp(-0.1 * age)              // 时间衰减因子

				friendPostScores[postID] += 0.8 * timeDecay
			}
		}

		// 好友喜欢的内容
		for postID, weight := range sn.UserPostMatrix[friendID] {
			if !interactedPosts[postID] {
				friendPostScores[postID] += 0.5 * weight
			}
		}
	}

	// 根据用户兴趣计算内容相关性
	interestPostScores := make(map[int]float64)

	for postID, post := range sn.Posts {
		if !interactedPosts[postID] {
			// 计算内容与用户兴趣的匹配度
			matchScore := 0.0

			// 根据标签计算匹配度
			for _, tag := range post.Tags {
				if weight, ok := user.Interests[tag]; ok {
					matchScore += weight
				}
			}

			interestPostScores[postID] = matchScore
		}
	}

	// 结合两种推荐策略的得分
	combinedScores := make(map[int]float64)

	for postID, friendScore := range friendPostScores {
		interestScore := interestPostScores[postID]
		// 在实际应用中，可以根据用户行为数据调整权重
		combinedScores[postID] = 0.7*friendScore + 0.3*interestScore
	}

	// 将未被好友交互但与兴趣相关的内容也纳入考虑
	for postID, interestScore := range interestPostScores {
		if _, ok := combinedScores[postID]; !ok {
			combinedScores[postID] = 0.3 * interestScore
		}
	}

	// 填充优先队列
	for postID, score := range combinedScores {
		if score > 0 {
			heap.Push(&pq, &RecommendationItem{
				ID:    postID,
				Score: score,
			})
		}
	}

	// 获取前count个推荐结果
	result := make([]*RecommendationItem, 0, min(count, pq.Len()))
	for i := 0; i < count && pq.Len() > 0; i++ {
		item := heap.Pop(&pq).(*RecommendationItem)
		result = append(result, item)
	}

	return result, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 辅助函数：创建演示用的社交网络数据
func createDemoSocialNetwork() *SocialNetwork {
	sn := NewSocialNetwork()

	// 创建用户
	interests := []string{"科技", "体育", "音乐", "电影", "旅游", "美食", "健身", "游戏", "汽车", "时尚"}

	for i := 1; i <= 20; i++ {
		// 为每个用户随机分配3-5个兴趣爱好
		userInterests := make(map[string]float64)
		numInterests := 3 + rand.Intn(3)
		interestsCopy := make([]string, len(interests))
		copy(interestsCopy, interests)
		rand.Shuffle(len(interestsCopy), func(i, j int) {
			interestsCopy[i], interestsCopy[j] = interestsCopy[j], interestsCopy[i]
		})

		for j := 0; j < numInterests; j++ {
			userInterests[interestsCopy[j]] = 0.5 + rand.Float64()*0.5 // 随机的兴趣程度
		}

		user := &User{
			ID:        i,
			Name:      fmt.Sprintf("用户%d", i),
			Interests: userInterests,
			Friends:   make(map[int]bool),
		}

		sn.AddUser(user)
	}

	// 创建社交关系（随机生成）
	for i := 1; i <= 20; i++ {
		// 每个用户随机添加3-7个好友
		numFriends := 3 + rand.Intn(5)
		for j := 0; j < numFriends; j++ {
			friendID := 1 + rand.Intn(20)
			if friendID != i && !sn.Users[i].Friends[friendID] {
				sn.AddFriendship(i, friendID)
			}
		}
	}

	// 创建内容
	for i := 1; i <= 50; i++ {
		// 随机选择1-3个标签
		numTags := 1 + rand.Intn(3)
		postTags := make([]string, 0, numTags)
		interestsCopy := make([]string, len(interests))
		copy(interestsCopy, interests)
		rand.Shuffle(len(interestsCopy), func(i, j int) {
			interestsCopy[i], interestsCopy[j] = interestsCopy[j], interestsCopy[i]
		})

		for j := 0; j < numTags; j++ {
			postTags = append(postTags, interestsCopy[j])
		}

		// 随机选择作者
		authorID := 1 + rand.Intn(20)

		// 创建随机的发布时间（过去30天内）
		randomDaysAgo := rand.Intn(30)
		postTime := time.Now().Add(-time.Duration(randomDaysAgo) * 24 * time.Hour)

		post := &Post{
			ID:        i,
			AuthorID:  authorID,
			Title:     fmt.Sprintf("内容 #%d", i),
			Content:   fmt.Sprintf("这是内容 #%d 的正文，由用户 %d 发布。", i, authorID),
			Tags:      postTags,
			Timestamp: postTime,
			Likes:     make(map[int]bool),
		}

		sn.AddPost(post)
	}

	// 创建用户与内容的交互
	for i := 1; i <= 20; i++ {
		// 每个用户随机点赞5-15个内容
		numLikes := 5 + rand.Intn(11)
		for j := 0; j < numLikes; j++ {
			postID := 1 + rand.Intn(50)
			sn.AddInteraction(i, postID, 1.0) // 1.0表示点赞
		}
	}

	return sn
}

// 场景示例：社交网络推荐系统演示
func SocialRecommendationDemo() {
	fmt.Println("社交网络推荐系统示例:")

	// 设置随机种子
	rand.Seed(time.Now().UnixNano())

	// 创建演示用的社交网络
	sn := createDemoSocialNetwork()

	// 选择一个目标用户
	targetUserID := 1 + rand.Intn(20)
	targetUser := sn.Users[targetUserID]

	fmt.Printf("\n为用户 %s (ID: %d) 生成推荐:\n", targetUser.Name, targetUser.ID)

	// 显示用户信息
	fmt.Printf("\n用户信息:\n")
	fmt.Printf("兴趣: ")

	// 对兴趣按权重排序
	type interestPair struct {
		interest string
		weight   float64
	}

	interestSlice := make([]interestPair, 0, len(targetUser.Interests))
	for interest, weight := range targetUser.Interests {
		interestSlice = append(interestSlice, interestPair{interest, weight})
	}

	sort.Slice(interestSlice, func(i, j int) bool {
		return interestSlice[i].weight > interestSlice[j].weight
	})

	for i, pair := range interestSlice {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%s (%.1f)", pair.interest, pair.weight)
	}
	fmt.Println()

	// 显示当前好友
	fmt.Printf("当前好友: ")
	friendIDs := make([]int, 0, len(targetUser.Friends))
	for friendID := range targetUser.Friends {
		friendIDs = append(friendIDs, friendID)
	}
	sort.Ints(friendIDs)

	for i, friendID := range friendIDs {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%s (ID: %d)", sn.Users[friendID].Name, friendID)
	}
	fmt.Println()

	// 生成好友推荐
	fmt.Printf("\n推荐好友:\n")
	friendRecs, err := sn.RecommendFriends(targetUserID, 5)
	if err != nil {
		fmt.Printf("推荐好友时出错: %v\n", err)
	} else {
		for i, rec := range friendRecs {
			recUser := sn.Users[rec.ID]
			fmt.Printf("%d. %s (ID: %d) - 相似度得分: %.2f\n", i+1, recUser.Name, recUser.ID, rec.Score)

			// 显示推荐原因
			fmt.Printf("   推荐原因: ")

			// 计算共同好友
			commonFriends := make([]int, 0)
			for friendID := range targetUser.Friends {
				if sn.Users[rec.ID].Friends[friendID] {
					commonFriends = append(commonFriends, friendID)
				}
			}

			// 计算共同兴趣
			commonInterests := make([]string, 0)
			for interest := range targetUser.Interests {
				if _, ok := recUser.Interests[interest]; ok {
					commonInterests = append(commonInterests, interest)
				}
			}

			reasons := make([]string, 0)
			if len(commonFriends) > 0 {
				friendNames := make([]string, 0, min(3, len(commonFriends)))
				for i := 0; i < min(3, len(commonFriends)); i++ {
					friendNames = append(friendNames, sn.Users[commonFriends[i]].Name)
				}
				reason := fmt.Sprintf("有%d个共同好友", len(commonFriends))
				if len(friendNames) > 0 {
					reason += fmt.Sprintf(" (包括 %s)", joinStrings(friendNames, ", "))
				}
				reasons = append(reasons, reason)
			}

			if len(commonInterests) > 0 {
				reason := fmt.Sprintf("有%d个共同兴趣", len(commonInterests))
				if len(commonInterests) > 0 {
					reason += fmt.Sprintf(" (包括 %s)", joinStrings(commonInterests[:min(3, len(commonInterests))], ", "))
				}
				reasons = append(reasons, reason)
			}

			fmt.Println(joinStrings(reasons, "; "))
		}
	}

	// 生成内容推荐
	fmt.Printf("\n推荐内容:\n")
	postRecs, err := sn.RecommendPosts(targetUserID, 5)
	if err != nil {
		fmt.Printf("推荐内容时出错: %v\n", err)
	} else {
		for i, rec := range postRecs {
			post := sn.Posts[rec.ID]
			fmt.Printf("%d. %s (ID: %d) - 推荐得分: %.2f\n", i+1, post.Title, post.ID, rec.Score)
			fmt.Printf("   标签: %s\n", joinStrings(post.Tags, ", "))
			fmt.Printf("   作者: %s (ID: %d)\n", sn.Users[post.AuthorID].Name, post.AuthorID)

			// 显示推荐原因
			fmt.Printf("   推荐原因: ")

			reasons := make([]string, 0)

			// 判断作者是否是好友
			if targetUser.Friends[post.AuthorID] {
				reasons = append(reasons, fmt.Sprintf("由你的好友 %s 发布", sn.Users[post.AuthorID].Name))
			}

			// 检查好友是否喜欢这篇内容
			friendsWhoLike := make([]int, 0)
			for friendID := range targetUser.Friends {
				if post.Likes[friendID] {
					friendsWhoLike = append(friendsWhoLike, friendID)
				}
			}

			if len(friendsWhoLike) > 0 {
				friendNames := make([]string, 0, min(2, len(friendsWhoLike)))
				for i := 0; i < min(2, len(friendsWhoLike)); i++ {
					friendNames = append(friendNames, sn.Users[friendsWhoLike[i]].Name)
				}
				reason := fmt.Sprintf("%d个好友喜欢这篇内容", len(friendsWhoLike))
				if len(friendNames) > 0 {
					reason += fmt.Sprintf(" (包括 %s)", joinStrings(friendNames, ", "))
				}
				reasons = append(reasons, reason)
			}

			// 检查是否与用户兴趣匹配
			matchingTags := make([]string, 0)
			for _, tag := range post.Tags {
				if _, ok := targetUser.Interests[tag]; ok {
					matchingTags = append(matchingTags, tag)
				}
			}

			if len(matchingTags) > 0 {
				reasons = append(reasons, fmt.Sprintf("与你的兴趣 %s 匹配", joinStrings(matchingTags, ", ")))
			}

			if len(reasons) == 0 {
				reasons = append(reasons, "最近热门内容")
			}

			fmt.Println(joinStrings(reasons, "; "))
		}
	}
}

// 辅助函数：连接字符串
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
