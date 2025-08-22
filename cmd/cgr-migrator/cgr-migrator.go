/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/migrator"
	"github.com/cgrates/cgrates/utils"
)

var (
	cgrMigratorFlags = flag.NewFlagSet(utils.CgrMigrator, flag.ContinueOnError)

	sameDataDB bool
	dmIN       = make(map[string]migrator.MigratorDataDB)
	dmOUT      = make(map[string]migrator.MigratorDataDB)
	err        error
	dfltCfg    = config.NewDefaultCGRConfig()
	cfgPath    = cgrMigratorFlags.String(utils.CfgPathCgr, utils.EmptyString,
		"Configuration directory path.")
	printConfig = cgrMigratorFlags.Bool(utils.PrintCfgCgr, false, "Print the configuration object in JSON format")
	exec        = cgrMigratorFlags.String(utils.ExecCgr, utils.EmptyString, "fire up automatic migration "+
		"<*set_versions|*cost_details|*accounts|*actions|*action_triggers|*action_plans|*shared_groups|*filters|*datadb>")
	version = cgrMigratorFlags.Bool(utils.VersionCgr, false, "prints the application version")

	inDBDataEncoding = cgrMigratorFlags.String(utils.DBDataEncodingCfg, dfltCfg.GeneralCfg().DBDataEncoding,
		"the encoding used to store object Data in strings")
	dbRedisMaxConns = cgrMigratorFlags.Int(utils.RedisMaxConnsCfg, dfltCfg.DataDbCfg().Opts.RedisMaxConns,
		"The connection pool size")
	dbRedisConnectAttempts = cgrMigratorFlags.Int(utils.RedisConnectAttemptsCfg, dfltCfg.DataDbCfg().Opts.RedisConnectAttempts,
		"The maximum amount of dial attempts")
	inDataDBRedisSentinel = cgrMigratorFlags.String(utils.RedisSentinelNameCfg, dfltCfg.DataDbCfg().Opts.RedisSentinel,
		"the name of redis sentinel")
	dbRedisCluster = cgrMigratorFlags.Bool(utils.RedisClusterCfg, false,
		"Is the redis datadb a cluster")
	dbRedisClusterSync = cgrMigratorFlags.Duration(utils.RedisClusterSyncCfg, dfltCfg.DataDbCfg().Opts.RedisClusterSync,
		"The sync interval for the redis cluster")
	dbRedisClusterDownDelay = cgrMigratorFlags.Duration(utils.RedisClusterOnDownDelayCfg, dfltCfg.DataDbCfg().Opts.RedisClusterOndownDelay,
		"The delay before executing the commands if the redis cluster is in the CLUSTERDOWN state")
	dbRedisConnectTimeout = cgrMigratorFlags.Duration(utils.RedisConnectTimeoutCfg, dfltCfg.DataDbCfg().Opts.RedisConnectTimeout,
		"The amount of wait time until timeout for a connection attempt")
	dbRedisReadTimeout = cgrMigratorFlags.Duration(utils.RedisReadTimeoutCfg, dfltCfg.DataDbCfg().Opts.RedisReadTimeout,
		"The amount of wait time until timeout for reading operations")
	dbRedisWriteTimeout = cgrMigratorFlags.Duration(utils.RedisWriteTimeoutCfg, dfltCfg.DataDbCfg().Opts.RedisWriteTimeout,
		"The amount of wait time until timeout for writing operations")
	dbRedisPoolPipelineWindow = cgrMigratorFlags.Duration(utils.RedisPoolPipelineWindowCfg, dfltCfg.DataDbCfg().Opts.RedisPoolPipelineWindow,
		"Duration after which internal pipelines are flushed. Zero disables implicit pipelining.")
	dbRedisPoolPipelineLimit = cgrMigratorFlags.Int(utils.RedisPoolPipelineLimitCfg, dfltCfg.DataDbCfg().Opts.RedisPoolPipelineLimit,
		"Maximum number of commands that can be pipelined before flushing. Zero means no limit.")
	dbRedisTls               = cgrMigratorFlags.Bool(utils.RedisTLSCfg, false, "Enable TLS when connecting to Redis")
	dbRedisClientCertificate = cgrMigratorFlags.String(utils.RedisClientCertificateCfg, utils.EmptyString, "Path to the client certificate")
	dbRedisClientKey         = cgrMigratorFlags.String(utils.RedisClientKeyCfg, utils.EmptyString, "Path to the client key")
	dbRedisCACertificate     = cgrMigratorFlags.String(utils.RedisCACertificateCfg, utils.EmptyString, "Path to the CA certificate")
	dbQueryTimeout           = cgrMigratorFlags.Duration(utils.MongoQueryTimeoutCfg, dfltCfg.DataDbCfg().Opts.MongoQueryTimeout,
		"The timeout for queries")
	dbMongoConnScheme = cgrMigratorFlags.String(utils.MongoConnSchemeCfg, dfltCfg.DataDbCfg().Opts.MongoConnScheme,
		"Scheme for MongoDB connection <mongodb|mongodb+srv>")

	outDBDataEncoding = cgrMigratorFlags.String(utils.OutDataDBEncodingCfg, utils.MetaDataDB,
		"the encoding used to store object Data in strings in move mode")
	outDataDBRedisSentinel = cgrMigratorFlags.String(utils.OutDataDBRedisSentinel, utils.MetaDataDB,
		"the name of redis sentinel")
	dryRun = cgrMigratorFlags.Bool(utils.DryRunCfg, false,
		"parse loaded data for consistency and errors, without storing it")
	verbose = cgrMigratorFlags.Bool(utils.VerboseCgr, false, "enable detailed verbose logging output")
)

func main() {
	if err := cgrMigratorFlags.Parse(os.Args[1:]); err != nil {
		return
	}
	if *version {
		if rcv, err := utils.GetCGRVersion(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(rcv)
		}
		return
	}

	mgrCfg := dfltCfg
	if *cfgPath != utils.EmptyString {
		if mgrCfg, err = config.NewCGRConfigFromPath(context.Background(), *cfgPath); err != nil {
			log.Fatalf("error loading config file %s", err.Error())
		}
		if mgrCfg.ConfigDBCfg().Type != utils.MetaInternal {
			d, err := engine.NewDataDBConn(mgrCfg.ConfigDBCfg().Type,
				mgrCfg.ConfigDBCfg().Host, mgrCfg.ConfigDBCfg().Port,
				mgrCfg.ConfigDBCfg().Name, mgrCfg.ConfigDBCfg().User,
				mgrCfg.ConfigDBCfg().Password, mgrCfg.GeneralCfg().DBDataEncoding,
				mgrCfg.ConfigDBCfg().Opts, nil)
			if err != nil { // Cannot configure getter database, show stopper
				utils.Logger.Crit(fmt.Sprintf("Could not configure configDB: %s exiting!", err))
				return
			}
			if err = mgrCfg.LoadFromDB(context.Background(), d); err != nil {
				log.Fatalf("Could not parse config: <%s>", err.Error())
				return
			}
		}
		config.SetCgrConfig(mgrCfg)
	}

	if *inDBDataEncoding != dfltCfg.GeneralCfg().DBDataEncoding {
		mgrCfg.GeneralCfg().DBDataEncoding = *inDBDataEncoding
	}
	if *dbRedisMaxConns != dfltCfg.DataDbCfg().Opts.RedisMaxConns {
		mgrCfg.DataDbCfg().Opts.RedisMaxConns = *dbRedisMaxConns
	}
	if *dbRedisConnectAttempts != dfltCfg.DataDbCfg().Opts.RedisConnectAttempts {
		mgrCfg.DataDbCfg().Opts.RedisConnectAttempts = *dbRedisConnectAttempts
	}
	if *inDataDBRedisSentinel != dfltCfg.DataDbCfg().Opts.RedisSentinel {
		mgrCfg.DataDbCfg().Opts.RedisSentinel = *inDataDBRedisSentinel
	}
	if *dbRedisCluster != dfltCfg.DataDbCfg().Opts.RedisCluster {
		mgrCfg.DataDbCfg().Opts.RedisCluster = *dbRedisCluster
	}
	if *dbRedisClusterSync != dfltCfg.DataDbCfg().Opts.RedisClusterSync {
		mgrCfg.DataDbCfg().Opts.RedisClusterSync = *dbRedisClusterSync
	}
	if *dbRedisClusterDownDelay != dfltCfg.DataDbCfg().Opts.RedisClusterOndownDelay {
		mgrCfg.DataDbCfg().Opts.RedisClusterOndownDelay = *dbRedisClusterDownDelay
	}
	if *dbRedisConnectTimeout != dfltCfg.DataDbCfg().Opts.RedisConnectTimeout {
		mgrCfg.DataDbCfg().Opts.RedisConnectTimeout = *dbRedisConnectTimeout
	}
	if *dbRedisReadTimeout != dfltCfg.DataDbCfg().Opts.RedisReadTimeout {
		mgrCfg.DataDbCfg().Opts.RedisReadTimeout = *dbRedisReadTimeout
	}
	if *dbRedisWriteTimeout != dfltCfg.DataDbCfg().Opts.RedisWriteTimeout {
		mgrCfg.DataDbCfg().Opts.RedisWriteTimeout = *dbRedisWriteTimeout
	}
	if *dbRedisPoolPipelineWindow != dfltCfg.DataDbCfg().Opts.RedisPoolPipelineWindow {
		mgrCfg.DataDbCfg().Opts.RedisPoolPipelineWindow = *dbRedisPoolPipelineWindow
	}
	if *dbRedisPoolPipelineLimit != dfltCfg.DataDbCfg().Opts.RedisPoolPipelineLimit {
		mgrCfg.DataDbCfg().Opts.RedisPoolPipelineLimit = *dbRedisPoolPipelineLimit
	}
	if *dbRedisTls != dfltCfg.DataDbCfg().Opts.RedisTLS {
		mgrCfg.DataDbCfg().Opts.RedisTLS = *dbRedisTls
	}
	if *dbRedisClientCertificate != dfltCfg.DataDbCfg().Opts.RedisClientCertificate {
		mgrCfg.DataDbCfg().Opts.RedisClientCertificate = *dbRedisClientCertificate
	}
	if *dbRedisClientKey != dfltCfg.DataDbCfg().Opts.RedisClientKey {
		mgrCfg.DataDbCfg().Opts.RedisClientKey = *dbRedisClientKey
	}
	if *dbRedisCACertificate != dfltCfg.DataDbCfg().Opts.RedisCACertificate {
		mgrCfg.DataDbCfg().Opts.RedisCACertificate = *dbRedisCACertificate
	}
	if *dbQueryTimeout != dfltCfg.DataDbCfg().Opts.MongoQueryTimeout {
		mgrCfg.DataDbCfg().Opts.MongoQueryTimeout = *dbQueryTimeout
	}
	if *dbMongoConnScheme != dfltCfg.DataDbCfg().Opts.MongoConnScheme {
		mgrCfg.DataDbCfg().Opts.MongoConnScheme = *dbMongoConnScheme
	}

	// outDataDB
	if *outDBDataEncoding == utils.MetaDataDB {
		if dfltCfg.MigratorCgrCfg().OutDataDBEncoding == mgrCfg.MigratorCgrCfg().OutDataDBEncoding {
			mgrCfg.MigratorCgrCfg().OutDataDBEncoding = mgrCfg.GeneralCfg().DBDataEncoding
		}
	} else {
		mgrCfg.MigratorCgrCfg().OutDataDBEncoding = *outDBDataEncoding
	}
	if *outDataDBRedisSentinel == utils.MetaDataDB {
		if dfltCfg.MigratorCgrCfg().OutDataDBOpts.RedisSentinel == mgrCfg.MigratorCgrCfg().OutDataDBOpts.RedisSentinel {
			mgrCfg.MigratorCgrCfg().OutDataDBOpts.RedisSentinel = dfltCfg.DataDbCfg().Opts.RedisSentinel
		}
	} else {
		mgrCfg.MigratorCgrCfg().OutDataDBOpts.RedisSentinel = *outDataDBRedisSentinel
	}

	outDbIdList := []string{} // will be populated with DataDB conn ids from migrator InItems
	// gather all db conns that will be used as OutDataDB using InItems datadb_ids
	for _, item := range mgrCfg.MigratorCgrCfg().InItems {
		if !slices.Contains(outDbIdList, item.DataDBID) {
			outDbIdList = append(outDbIdList, item.DataDBID)
		}
	}

	inDbIdList := []string{} // will be populated with DataDB conn ids
	// gather all db conns that will be used as OutDataDB using InItems datadb_ids
	for _, item := range mgrCfg.MigratorCgrCfg().InItems {
		if !slices.Contains(inDbIdList, item.DataDBID) {
			inDbIdList = append(inDbIdList, item.DataDBID)
		}
	}

	// order and compare the datadbIDs, if they are the same we only need to compare encodings
	// to know if DBs are the same or not
	if utils.EqualUnorderedStringSlices(inDbIdList, outDbIdList) {
		sameDataDB = mgrCfg.MigratorCgrCfg().OutDataDBEncoding == mgrCfg.GeneralCfg().DBDataEncoding
	}

	if dmIN, err = migrator.NewMigratorDataDBs(inDbIdList, mgrCfg.GeneralCfg().DBDataEncoding, mgrCfg); err != nil {
		log.Fatal(err)
	}

	if *printConfig {
		cfgJSON := utils.ToIJSON(mgrCfg.AsMapInterface())
		log.Printf("Configuration loaded from %q:\n%s", *cfgPath, cfgJSON)
	}

	if sameDataDB {
		dmOUT = dmIN
	} else {
		if dmOUT, err = migrator.NewMigratorDataDBs(outDbIdList, mgrCfg.MigratorCgrCfg().OutDataDBEncoding, mgrCfg); err != nil {
			log.Fatal(err)
		}
	}

	m, err := migrator.NewMigrator(mgrCfg.DataDbCfg(), dmIN, dmOUT, *dryRun, sameDataDB)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()
	config.SetCgrConfig(mgrCfg)
	if exec != nil && *exec != utils.EmptyString { // Run migrator
		migrstats := make(map[string]int)
		mig := strings.Split(*exec, utils.FieldsSep)
		err, migrstats = m.Migrate(mig)
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			log.Printf("Data migrated: %+v", migrstats)
		}
		return
	}

}
