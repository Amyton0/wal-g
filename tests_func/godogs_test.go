package functests

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/spf13/pflag"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/tests_func/utils"
)

type TestOpts struct {
	test          bool
	clean         bool
	stop          bool
	debug         bool
	featurePrefix string
	database      string
}

var (
	godogOpts = godog.Options{
		Output:        colors.Colored(os.Stdout),
		Format:        "pretty",
		StopOnFailure: true,
		Strict:        true,
	}

	testOpts = TestOpts{}

	databases = map[string]bool{
		"mongodb": true,
		"redis":   true,
	}
)

func init() {
	pflag.BoolVar(&testOpts.test, "tf.test", true, "run tests")
	pflag.BoolVar(&testOpts.stop, "tf.stop", true, "shutdown test environment")
	pflag.BoolVar(&testOpts.clean, "tf.clean", true, "delete test environment")
	pflag.BoolVar(&testOpts.debug, "tf.debug", false, "enable debug logging")
	pflag.StringVar(&testOpts.featurePrefix, "tf.featurePrefix", "", "features prefix")
	pflag.StringVar(&testOpts.database, "tf.database", "", "database name [mongodb|redis]")

	godog.BindCommandLineFlags("godog.", &godogOpts)
}

func parseArgs() {
	pflag.Parse()

	if _, ok := databases[testOpts.database]; !ok {
		tracelog.ErrorLogger.Fatalf("Database '%s' is not valid, please provide test database -tf.database=dbname\n",
			testOpts.database)
	}
}

func TestMain(m *testing.M) {
	parseArgs()

	if testOpts.debug {
		err := tracelog.UpdateLogLevel(tracelog.DevelLogLevel)
		tracelog.ErrorLogger.FatalOnError(err)
	}

	tctx, err := CreateTestContex(testOpts.database)
	tracelog.ErrorLogger.FatalOnError(err)

	status := 0

	if testOpts.test {
		godogOpts.Paths, err = utils.FindFeaturePaths(testOpts.database, testOpts.featurePrefix)
		tracelog.ErrorLogger.FatalOnError(err)

		tracelog.InfoLogger.Printf("Starting testing environment: mongodb %s with features: %v",
			tctx.Version.Full, godogOpts.Paths)

		suite := godog.TestSuite{
			Name: "godogs",
			TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
				ctx.BeforeSuite(tctx.LoadEnv)
			},
			ScenarioInitializer: func(ctx *godog.ScenarioContext) {
				setupCommonSteps(ctx, tctx)
				setupMongodbSteps(ctx, tctx)
				setupRedisSteps(ctx, tctx)
			},
			Options: &godogOpts,
		}
		status = suite.Run()
	}

	if testOpts.stop {
		err = tctx.StopEnv()
		tracelog.ErrorLogger.FatalOnError(err)
	}

	if testOpts.clean {
		err = tctx.CleanEnv()
		tracelog.ErrorLogger.FatalOnError(err)
	}

	os.Exit(status)
}
