package repository

import (
	"reflect"
	"testing"

	"collector-backend/auth"
	"collector-backend/models"
)

// TestPublicFileTagsAndInteractions 验证标签检索、唯一点赞、评论和公开图片边界。
func TestPublicFileTagsAndInteractions(t *testing.T) {
	// store 使用测试临时目录中的独立 SQLite，不接触业务数据库。
	store, _ := openTempStore(t)
	defer store.db.Close()

	// canLogin 表示互动测试账号允许建立登录会话。
	canLogin := true
	// firstUser、message 保存第一个互动账号及创建结果。
	firstUser, message := store.CreateUser(models.UserRequest{
		Username: "public-interaction-1",
		Name:     "互动用户一",
		Role:     "普通用户",
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create first interaction user: %s", message)
	}
	// secondUser、message 保存第二个互动账号及创建结果。
	secondUser, message := store.CreateUser(models.UserRequest{
		Username: "public-interaction-2",
		Name:     "互动用户二",
		Role:     "普通用户",
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create second interaction user: %s", message)
	}

	// publicImage 保存带规范化标签的公开图片。
	publicImage := store.CreateFile(models.ManagedFile{
		DisplayName:  "测试图片.png",
		OriginalName: "original.png",
		Category:     "插画",
		Description:  "公开图片",
		Tags:         []string{"#风景", "夜景", "风景", "abcdefghijklmnopqrstuvwxy"},
		ContentType:  "image/png",
		StorageName:  "public-image.png",
		OwnerID:      firstUser.ID,
	})
	if publicImage.ID == 0 {
		t.Fatal("create public image failed")
	}
	// expectedTags 保存去前缀、去重和长度截断后的标签。
	expectedTags := []string{"风景", "夜景", "abcdefghijklmnopqrstuvwx"}
	if !reflect.DeepEqual(publicImage.Tags, expectedTags) {
		t.Fatalf("unexpected normalized tags: %+v", publicImage.Tags)
	}

	// matchedImages、matchedTotal 保存按标签搜索得到的公开图片。
	matchedImages, matchedTotal, _ := store.LatestPublicImages("夜景", "", 1, 24, false)
	if matchedTotal != 1 || len(matchedImages) != 1 || matchedImages[0].ID != publicImage.ID || !reflect.DeepEqual(matchedImages[0].Tags, expectedTags) {
		t.Fatalf("tag search did not return public image: total=%d items=%+v", matchedTotal, matchedImages)
	}

	// appendedTags、added、found 保存门户用户追加标签后的权威结果。
	appendedTags, added, found := store.AppendPublicFileTag(publicImage.ID, firstUser.ID, "#城市", false)
	if !found || !added || !reflect.DeepEqual(appendedTags, []string{"风景", "夜景", "abcdefghijklmnopqrstuvwx", "城市"}) {
		t.Fatalf("append public tag failed: found=%v added=%v tags=%+v", found, added, appendedTags)
	}
	// duplicateTags、added、found 保存重复追加时不改写已有标签的结果。
	duplicateTags, added, found := store.AppendPublicFileTag(publicImage.ID, secondUser.ID, "城市", false)
	if !found || added || !reflect.DeepEqual(duplicateTags, appendedTags) {
		t.Fatalf("duplicate public tag changed tags: found=%v added=%v tags=%+v", found, added, duplicateTags)
	}
	// fillTagIndex 表示用于填满十二个标签上限的测试标签序号。
	for fillTagIndex := 1; fillTagIndex <= 8; fillTagIndex++ {
		// fillTags、fillAdded、fillFound 保存当前补充标签的写入结果。
		fillTags, fillAdded, fillFound := store.AppendPublicFileTag(publicImage.ID, firstUser.ID, "补充标签"+string(rune('A'+fillTagIndex-1)), false)
		if !fillFound || !fillAdded || len(fillTags) != 4+fillTagIndex {
			t.Fatalf("fill public tags failed at %d: found=%v added=%v tags=%+v", fillTagIndex, fillFound, fillAdded, fillTags)
		}
	}
	// limitedTags、added、found 保存达到数量上限后拒绝覆盖旧标签的结果。
	limitedTags, added, found := store.AppendPublicFileTag(publicImage.ID, firstUser.ID, "第十三个标签", false)
	if !found || added || len(limitedTags) != 12 {
		t.Fatalf("public tag limit failed: found=%v added=%v tags=%+v", found, added, limitedTags)
	}

	// firstLike 保存第一个账号首次点赞后的权威互动状态。
	firstLike, updated := store.TogglePublicFileLike(publicImage.ID, firstUser.ID, false)
	if !updated || firstLike.LikeCount != 1 || !firstLike.LikedByCurrentUser {
		t.Fatalf("first like failed: updated=%v interaction=%+v", updated, firstLike)
	}
	// secondLike 保存第二个账号点赞后的独立互动状态。
	secondLike, updated := store.TogglePublicFileLike(publicImage.ID, secondUser.ID, false)
	if !updated || secondLike.LikeCount != 2 || !secondLike.LikedByCurrentUser {
		t.Fatalf("second user like failed: updated=%v interaction=%+v", updated, secondLike)
	}
	// firstUnlike 保存第一个账号再次点击后取消点赞的状态。
	firstUnlike, updated := store.TogglePublicFileLike(publicImage.ID, firstUser.ID, false)
	if !updated || firstUnlike.LikeCount != 1 || firstUnlike.LikedByCurrentUser {
		t.Fatalf("unlike failed: updated=%v interaction=%+v", updated, firstUnlike)
	}

	// createdComment 保存第一个账号发送的纯文本评论。
	createdComment, created := store.CreatePublicFileComment(publicImage.ID, firstUser.ID, "第一条评论", false)
	if !created || createdComment.ID == 0 || createdComment.UserName != firstUser.Name || createdComment.Content != "第一条评论" {
		t.Fatalf("create comment failed: created=%v comment=%+v", created, createdComment)
	}
	// interaction 保存匿名读取的点赞数量和评论列表。
	interaction, found := store.GetPublicFileInteraction(publicImage.ID, 0, false)
	if !found || interaction.LikeCount != 1 || interaction.LikedByCurrentUser || len(interaction.Comments) != 1 || interaction.Comments[0].ID != createdComment.ID {
		t.Fatalf("unexpected anonymous interaction: found=%v interaction=%+v", found, interaction)
	}

	// privateImage 保存不允许公开互动的私密图片。
	privateImage := store.CreateFile(models.ManagedFile{
		DisplayName: "私密图片.png", OriginalName: "private.png", ContentType: "image/png", StorageName: "private.png", OwnerID: firstUser.ID, IsPrivate: true,
	})
	// publicDocument 保存公开但不是图片的文件。
	publicDocument := store.CreateFile(models.ManagedFile{
		DisplayName: "文档.txt", OriginalName: "document.txt", ContentType: "text/plain", StorageName: "document.txt", OwnerID: firstUser.ID,
	})
	if _, found := store.GetPublicFileInteraction(privateImage.ID, firstUser.ID, false); found {
		t.Fatal("private image interaction should not be public")
	}
	if _, updated := store.TogglePublicFileLike(publicDocument.ID, firstUser.ID, false); updated {
		t.Fatal("non-image file should not accept likes")
	}
	if _, created := store.CreatePublicFileComment(publicDocument.ID, firstUser.ID, "不应写入", false); created {
		t.Fatal("non-image file should not accept comments")
	}
	if _, added, found := store.AppendPublicFileTag(privateImage.ID, firstUser.ID, "不应写入", false); found || added {
		t.Fatal("private image should not accept public tags")
	}
	if _, added, found := store.AppendPublicFileTag(publicDocument.ID, firstUser.ID, "不应写入", false); found || added {
		t.Fatal("non-image file should not accept public tags")
	}
	if _, added, found := store.AppendPublicFileTag(publicImage.ID, 0, "不应写入", false); found || added {
		t.Fatal("anonymous user should not append public tags")
	}
}
