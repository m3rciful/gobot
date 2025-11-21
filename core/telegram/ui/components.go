package ui

import tele "gopkg.in/telebot.v4"

// NewSimpleArticleResult creates an ArticleResult with given ID, title and content.
func NewSimpleArticleResult(id, title, text string) *tele.ArticleResult {
	result := &tele.ArticleResult{
		Title: title,
		Text:  text,
	}
	result.SetResultID(id)
	return result
}
