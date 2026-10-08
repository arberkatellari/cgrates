// Copyright ITsysCOM GmbH
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

// // URFiltersFromReader used on URs Get
// type URFiltersFromReader struct {
// 	Tenant    string
// 	FilterIDs []string       // Filter for URs
// 	ReaderIDs []string       // ReaderIDs where the URs will be read from (ERs ReaderIDs)
// 	APIOpts   map[string]any // options for pagination
// }

// // URFiltersFromExporter used on URs Set or Remove
// type URFiltersFromExporter struct {
// 	URFilters   *utils.URFilters // URs which will overwrite an existing UR or set a new one using EEs
// 	ExporterIDs []string         // ExporterIDs where the URs will be exported to (EEs ExporterIDs)
// }

// // URFiltersFromDBConnID used on URs Get/Set/Remove
// type URFiltersFromDBConnID struct {
// 	URFilters *utils.URFilters // URs to Get/Set/Remove based on filters
// 	DBConnIDs []string         // DBConnIDs to use to query
// }

// // GetURs retrieves a list of URs matching the specified filters.
// func (admS AdminSv1) GetURs(ctx *context.Context, args *URFiltersFromReader, reply *[]*utils.UR) error {
// 	if args.Tenant == utils.EmptyString {
// 		args.Tenant = admS.cfg.GeneralCfg().DefaultTenant
// 	}
// 	fltrs, err := engine.GetFilters(ctx, args.FilterIDs, args.Tenant, admS.dm)
// 	if err != nil {
// 		return fmt.Errorf("preparing filters failed: %w", err)
// 	}
// 	readers := []*config.EventReaderCfg{} // holds all readers information used to get URs from
// 	for _, rdr := range admS.cfg.ERsCfg().Readers {
// 		if len(args.ReaderIDs) != 0 { // get readers from ERs from ReaderIDs
// 			if slices.Contains(args.ReaderIDs, rdr.ID) {
// 				if rdr.Type == utils.MetaCgrur {
// 					readers = append(readers, rdr)
// 				}
// 			}
// 		} else { // if ReaderIDs are not provided from parameters, get all *cgrur readers
// 			if rdr.Type == utils.MetaCgrur {
// 				readers = append(readers, rdr)
// 			}
// 		}
// 	}

// 	// try to get the URs from the selected readers
// 	for _, rdr := range readers {
// 		inURL := strings.TrimPrefix(rdr.SourcePath, utils.Meta)
// 		u, err := url.Parse(inURL)
// 		if err != nil {
// 			return err
// 		}
// 		password, _ := u.User.Password()

// 		dbname := utils.SQLDefaultDBName
// 		if rdr.Opts.SQLDBName != nil {
// 			dbname = *rdr.Opts.SQLDBName
// 		}
// 		ssl := utils.SQLDefaultPgSSLMode
// 		if rdr.Opts.PgSSLMode != nil {
// 			ssl = *rdr.Opts.PgSSLMode
// 		}

// 		rdr.tableName = utils.URsTBL
// 		if opts.SQLTableName != nil {
// 			rdr.tableName = *opts.SQLTableName
// 		}
// 		switch u.Scheme {
// 		case utils.MySQL:
// 			rdr.connString = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&loc=Local&parseTime=true&sql_mode='ALLOW_INVALID_DATES'",
// 				u.User.Username(), password, u.Hostname(), u.Port(), dbname)
// 		case utils.Postgres:
// 			rdr.connString = fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
// 				u.Hostname(), u.Port(), dbname, u.User.Username(), password, ssl)
// 		default:
// 			return fmt.Errorf("unknown dbType %s", u.Scheme)
// 		}
// 	}
// 	return
// }

// // SetURs sets or overwrites URs matching the specified filters.
// func (admS AdminSv1) SetURs(ctx *context.Context, args *URFiltersFromExporter, reply *string) (err error) {
// 	if args.URFilters.Tenant == utils.EmptyString {
// 		args.URFilters.Tenant = admS.cfg.GeneralCfg().DefaultTenant
// 	}
// 	fltrs, err := engine.GetFilters(ctx, args.URFilters.FilterIDs, args.URFilters.Tenant, admS.dm)
// 	if err != nil {
// 		return fmt.Errorf("preparing filters failed: %w", err)
// 	}
// 	if err := admS.dm.SetURs(ctx, fltrs); err != nil {
// 		return fmt.Errorf("removing URs failed: %w", err)
// 	}
// 	*reply = utils.OK
// 	return
// }

// // RemoveURs removes URs matching the specified filters.
// func (admS AdminSv1) RemoveURs(ctx *context.Context, args *URFiltersFromExporter, reply *string) (err error) {
// 	if args.URFilters.Tenant == utils.EmptyString {
// 		args.URFilters.Tenant = admS.cfg.GeneralCfg().DefaultTenant
// 	}
// 	fltrs, err := engine.GetFilters(ctx, args.URFilters.FilterIDs, args.URFilters.Tenant, admS.dm)
// 	if err != nil {
// 		return fmt.Errorf("preparing filters failed: %w", err)
// 	}
// 	if err := admS.dm.RemoveURs(ctx, fltrs); err != nil {
// 		return fmt.Errorf("removing URs failed: %w", err)
// 	}
// 	*reply = utils.OK
// 	return
// }

// // GetURs retrieves a list of URs matching the specified filters.
// func (admS AdminSv1) GetURsFromDBConnIDs(ctx *context.Context, args *URFiltersFromDBConnID, reply *[]*utils.UR) error {
// 	if args.URFilters.Tenant == utils.EmptyString {
// 		args.URFilters.Tenant = admS.cfg.GeneralCfg().DefaultTenant
// 	}
// 	fltrs, err := engine.GetFilters(ctx, args.URFilters.FilterIDs, args.URFilters.Tenant, admS.dm)
// 	if err != nil {
// 		return fmt.Errorf("preparing filters failed: %w", err)
// 	}
// 	dbConnIDs := make([]*config.DBConn, 0)
// 	if len(args.DBConnIDs) != 0 {
// 		for _, dbConnID := range args.DBConnIDs {
// 			if dbInfo, has := admS.cfg.DbCfg().DBConns[dbConnID]; has {
// 				dbConnIDs = append(dbConnIDs, dbInfo)
// 			}
// 		}
// 	} else {
// 		for _, dbInfo := range admS.cfg.DbCfg().DBConns {
// 			dbConnIDs = append(dbConnIDs, dbInfo)
// 		}
// 	}
// 	for _, db := range dbConnIDs {
// 		switch db.Type {
// 		case utils.MySQL, utils.MetaMySQL:

// 		case utils.Postgres, utils.MetaPostgres:
// 		default:
// 			return fmt.Errorf("Getting URs from db type <%v> not supported", db.Type)
// 		}
// 	}
// 	urs, err := admS.dm.GetURs(ctx, fltrs, args.URFilters.APIOpts)
// 	if err != nil {
// 		return fmt.Errorf("retrieving URs failed: %w", err)
// 	}
// 	*reply = urs
// 	return nil
// }

// // SetURs sets or overwrites URs matching the specified filters.
// func (admS AdminSv1) SetURsFromDBConnIDs(ctx *context.Context, args *URFiltersFromDBConnID, reply *string) (err error) {
// 	if args.URFilters.Tenant == utils.EmptyString {
// 		args.URFilters.Tenant = admS.cfg.GeneralCfg().DefaultTenant
// 	}
// 	fltrs, err := engine.GetFilters(ctx, args.URFilters.FilterIDs, args.URFilters.Tenant, admS.dm)
// 	if err != nil {
// 		return fmt.Errorf("preparing filters failed: %w", err)
// 	}
// 	if err := admS.dm.SetURs(ctx, fltrs); err != nil {
// 		return fmt.Errorf("removing URs failed: %w", err)
// 	}
// 	*reply = utils.OK
// 	return
// }

// // RemoveURs removes URs matching the specified filters.
// func (admS AdminSv1) RemoveURsFromDBConnIDs(ctx *context.Context, args *URFiltersFromDBConnID, reply *string) (err error) {
// 	if args.URFilters.Tenant == utils.EmptyString {
// 		args.URFilters.Tenant = admS.cfg.GeneralCfg().DefaultTenant
// 	}
// 	fltrs, err := engine.GetFilters(ctx, args.URFilters.FilterIDs, args.URFilters.Tenant, admS.dm)
// 	if err != nil {
// 		return fmt.Errorf("preparing filters failed: %w", err)
// 	}
// 	if err := admS.dm.RemoveURs(ctx, fltrs); err != nil {
// 		return fmt.Errorf("removing URs failed: %w", err)
// 	}
// 	*reply = utils.OK
// 	return
// }
