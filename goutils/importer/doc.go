package importer

//  有两个例子， 导入和导出
//
//  1. 导入
// func Import(ctx context.Context, request *http.Request) error {
//   // 将从附件从读记录， 附件可以是 excel 或 csv 文件
//   reader, closer, err := importer.ReadHTTP(ctx, request)
//   if err != nil {
//     return err
//   }
//   defer closer.Close()
//
//   return importer.Import(ctx, "", reader, func(context.Context) (importer.Row, error) {
//     record := &User{}
//     return importer.Row{
//       // Columns 为字段列表
//       Columns: []importer.Column{
//         importer.StrColumn([]string{"name", "用户"}, true,
//           func(ctx context.Context, origin, value string) error {
//             record.Name = value
//             return nil
//           }),
//         importer.StrColumn([]string{"sex", "性别"}, false,
//           func(ctx context.Context, origin, value string) error {
//             record.Sex = value
//             return nil
//           }),
//         importer.StrColumn([]string{"outer_inode_account", "外网Inode账号"}, false,
//           func(ctx context.Context, origin, value string) error {
//             record.OuterInodeAccount = value
//             return nil
//           }),
//         importer.StrColumn([]string{"outer_address", "外网IP"}, false,
//           func(ctx context.Context, origin, value string) error {
//             record.OuterAddress = value
//             return nil
//           }),
//
//          ......
//
//       },
//
//       // Commit 为一条记录讯完成后，进行处理
//       Commit: func(ctx context.Context) error {
//         old, err := records.users.FindByName(ctx, record.Name)
//         if err != nil && !errors.IsNotFound(err) {
//           return err
//         }
//         if old != nil {
//           if record.Sex != "" {
//             old.Sex = record.Sex
//           }
//           if record.Type != "" {
//             old.Type = record.Type
//           }
//
//           ......
//
//           _, err = records.users.UpdateByID(ctx, old.ID, old)
//           return err
//         }
//         _, err = records.users.Insert(ctx, record)
//         return err
//       },
//     }, nil
//   })
// }

// 2.导出的例子
// func Export(ctx context.Context, format string, inline bool, writer http.ResponseWriter) error {
//   return importer.WriteHTTP(ctx, "users", format, inline, writer,
//     importer.RecorderFunc(func(ctx context.Context) (importer.RecordIterator, []string, error) {
//       list, err := records.users.List(ctx)
//       if err != nil {
//         return nil, nil, err
//       }
//       titles := []string{
//         "部门",
//         "姓名",
//         "性别",
//         "类型",
//         "岗位",
//         "座机号",
//         "手机号",
//         "房间号",
//         "座位号",
//         "内网Inode帐号",
//         "内网IP",
//         "外网Inode帐号",
//         "外网IP",
//         "创建时间",
//         "更新时间",
//       }
//       departmentCache := map[int64]*XldwDepartment{}
//       index := 0
//
//       return importer.RecorderFuncIterator{
//         CloseFunc: func() error {
//           return nil
//         },
//         NextFunc: func(ctx context.Context) bool {
//           index++
//           return index < len(list)
//         },
//         ReadFunc: func(ctx context.Context) ([]string, error) {
//           department := departmentCache[list[index].DepartmentID]
//           if department == nil {
//             d, err := records.departments.FindByID(ctx, list[index].DepartmentID)
//             if err != nil {
//               return nil, err
//             }
//             departmentCache[list[index].DepartmentID] = d
//             department = d
//           }
//
//           return []string{
//             department.Name,
//             list[index].Name,
//             ToSexLabel(list[index].Sex),
//             list[index].Type,
//             list[index].Job,
//             list[index].Phone,
//             list[index].Mobile,
//             list[index].Room,
//             list[index].Seat,
//             list[index].InnerInodeAccount,
//             list[index].InnerAddress,
//             list[index].OuterInodeAccount,
//             list[index].OuterAddress,
//             formatTime(list[index].CreatedAt),
//             formatTime(list[index].UpdatedAt),
//           }, nil
//         },
//       }, titles, nil
//     }))
// }
