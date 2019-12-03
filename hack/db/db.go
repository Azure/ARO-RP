package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
)

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(ctx, env, uuid.NewV4(), "ARO")
	if err != nil {
		return err
	}

	doc, err := db.OpenShiftClusters.Get(api.Key(strings.ToLower(os.Args[1])))
	if err != nil {
		return err
	}

	h := &codec.JsonHandle{
		Indent: 4,
	}

	err = api.AddExtensions(&h.BasicHandle)
	if err != nil {
		return err
	}

	return codec.NewEncoder(os.Stdout, h).Encode(doc)
}

func main() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	log := logrus.NewEntry(logrus.StandardLogger())

	if err := run(context.Background(), log); err != nil {
		panic(err)
	}
}
