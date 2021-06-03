package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lucasew/pocket2kindle/epub"
	"github.com/lucasew/pocket2kindle/parse"
	"github.com/lucasew/pocket2kindle/pocket"
	p_api "github.com/motemen/go-pocket/api"
	"github.com/scorredoira/email"
)

type App struct {
    *log.Logger
    // options
    ArticleCount int
    DontDeleteIntermediates bool
    KindleEmail string // if not specified SMTP info will not be asked
    ArchiveBundled bool
    Timeout int
    // auth
    PocketRequestToken string
    PocketConsumerKey string
    SMTPServer string
    SMTPUser string
    SMTPPassword string
    // state
    processedPocketArticles []int
}

var (
    ErrExecutableNotFound = fmt.Errorf("executable not found")
)

func (a *App) StepConvertBook(ctx context.Context, epubFile string, mobiFile string) error {
    a.Printf("Converting epub file to mobi...")
    err := a.LookupBinary("ebook-convert")
    if err != nil {
        return err
    }
    cmd := exec.Command("ebook-convert", epubFile, mobiFile)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err = RunCommandContext(ctx, cmd)
    if errors.Is(err, ErrNonZeroExitCode) {
        return nil
    }
    return err
}

func (a *App) StepSendEmail(ctx context.Context, file string) error {
    if a.KindleEmail == "" {
        return nil
    }
    defer a.DeleteIntermediate(file)
    a.Printf("Sending converted file as attachment to email...")
    m := email.NewMessage("", "")
    m.From = mail.Address{Name: "p2k bot", Address: a.SMTPUser}
    m.To = []string{
        a.KindleEmail,
    }
    err := m.Attach(file)
    if err != nil {
        return err
    }
    auth := smtp.PlainAuth("", a.SMTPUser, a.SMTPPassword, strings.Split(a.SMTPServer, ":")[0])
    return email.Send(a.SMTPServer, auth, m)
}

func (a *App) GetFavUrl(articleId int) string {
    now := time.Now()
    favoriteUrl := "https://getpocket.com/v3/send?actions=%5B%7B%22action%22%3A%22favorite%22%2C%22time%22%3A[TIME]%2C%22item_id%22%3A[ITEM_ID]%7D%5D&access_token=[ACCESS_TOKEN]&consumer_key=[CONSUMER_KEY]"
    favoriteUrl = strings.ReplaceAll(favoriteUrl, "[ACCESS_TOKEN]", a.PocketRequestToken)
    favoriteUrl = strings.ReplaceAll(favoriteUrl, "[CONSUMER_KEY]", a.PocketConsumerKey)
    favoriteUrl = strings.ReplaceAll(favoriteUrl, "[TIME]", fmt.Sprintf("%d", now.Unix()))
    favoriteUrl = strings.ReplaceAll(favoriteUrl, "[ITEM_ID]", fmt.Sprintf("%d", articleId))
    return favoriteUrl
}

func (a *App) getPocketActor() pocket.PocketActor {
    return pocket.NewPocketActor(a.PocketConsumerKey, a.PocketRequestToken)
}

func (a *App) StepFetchArticles(ctx context.Context) ([]epub.EpubArticle, error) {
    var cancel func()
    ctx, cancel = context.WithCancel(ctx)
    defer cancel()

    const concurrentJobs = 4
    articles := make([]epub.EpubArticle, 0, a.ArticleCount)
    a.Printf("Starting article retriever...")
    retriever := a.getPocketActor().Retrieve(p_api.RetrieveOption{
        State: p_api.StateUnread,
        Count: a.ArticleCount,
        DetailType: p_api.DetailTypeComplete,
        Favorite: p_api.FavoriteFilterUnfavorited,
    })
    processedArticleChan := make(chan(epub.EpubArticle), 4)
    var processArticle func(ctx context.Context)
    processArticle = func(ctx context.Context) {
        articlePtr := retriever.Next(ctx)
        if articlePtr == nil {
            return
        }
        article := *articlePtr

        a.Printf("Got article '%s'", article.ResolvedTitle)
        parsedArticle, err := parser.Parse(ctx, article.ResolvedURL)
        if err != nil {
            a.Printf("Error when parsing article '%s': %s", article.ResolvedTitle, err)
            go processArticle(ctx)
            return
        }
        u, err := url.Parse(article.ResolvedURL)
        if err != nil {
            a.Printf("Can't parse url '%s': %s", article.ResolvedURL, err)
            go processArticle(ctx)
            return
        }
        epubArticle := epub.EpubArticle{
            Title: article.ResolvedTitle,
            URL: *u,
            Content: parsedArticle.Content,
            Actions: map[string]string{
                article.ResolvedURL: article.ResolvedURL,
                "Favorite": a.GetFavUrl(article.ItemID),
            },
        }
        select {
        case processedArticleChan <- epubArticle:
            a.Printf("Processed article '%s'!", article.ResolvedTitle)
            a.processedPocketArticles = append(a.processedPocketArticles, article.ItemID)
            go processArticle(ctx)
            return
        case <-ctx.Done():
            return
        }
    }
    for i := 0; i < concurrentJobs; i++ {
        go processArticle(ctx)
    }
    for i := 0; i < a.ArticleCount; i++ {
        select {
        case <-ctx.Done():
            break
        default:
            articles = append(articles, <-processedArticleChan)
        }
    }
    return articles, nil
}

func (a *App) Run(ctx context.Context) error {
    now := time.Now()
    generateContext, cancel := context.WithTimeout(ctx, time.Duration(a.Timeout)*time.Second)
    defer cancel()

    articles, err := a.StepFetchArticles(generateContext)
    if err != nil {
        a.Fatalf(err.Error())
    }

    epubOptions := epub.EpubOptions{
        Author: "Some machine, somewhere",
        Title: fmt.Sprintf("Pocket articles %d/%d/%d %d:%2.d", now.Day(), now.Month(), now.Year(), now.Hour(), now.Minute()),
    }
    epubFile := fmt.Sprintf("%s.epub", uuid.New())
    mobiFile := fmt.Sprintf("%s.mobi", uuid.New())
    a.Printf("Generating epub file...")
    err = epub.CreateEpub(articles, epubOptions, epubFile)
    if err != nil {
        return err
    }
    defer a.DeleteIntermediate(epubFile)
    err = a.StepConvertBook(ctx, epubFile, mobiFile)
    if err != nil {
        return err
    }
    err = a.StepSendEmail(ctx, mobiFile)
    if err != nil {
        return err
    }
    err = a.StepArchive()
    if err != nil {
        return err
    }
    return nil
}

func (a *App) StepArchive() error {
    if a.ArchiveBundled {
        log.Printf("Archiving bundled articles...")
        actions := make([]*pocket.Action, 0, len(a.processedPocketArticles))
        for _, id := range a.processedPocketArticles {
            actions = append(actions, pocket.NewArchiveAction(id))
        }
        return a.getPocketActor().Modify(actions...)
    }
    return nil
}
