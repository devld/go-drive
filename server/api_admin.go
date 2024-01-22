package server

import (
	"go-drive/common"
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/server/scheduled"
	"go-drive/server/search"
	"go-drive/storage"

	"github.com/gin-gonic/gin"
)

func InitAdminRoutes(
	r gin.IRouter,
	ch *registry.ComponentsHolder,
	config common.Config,
	bus event.Bus,
	runner task.Runner,
	jobExecutor *scheduled.JobExecutor,
	access *drive.Access,
	rootDrive *drive.RootDrive,
	search *search.Service,
	tokenStore types.TokenStore,
	optionsDAO *storage.OptionsDAO,
	userDAO *storage.UserDAO,
	groupDAO *storage.GroupDAO,
	driveDAO *storage.DriveDAO,
	driveDataDAO *storage.DriveDataDAO,
	permissionDAO *storage.PathPermissionDAO,
	pathMountDAO *storage.PathMountDAO,
	pathMetaDAO *storage.PathMetaDAO,
	scheduledDAO *storage.ScheduledDAO,
	fileBucketDAO *storage.FileBucketDAO) error {

	r = r.Group("/admin", TokenAuth(tokenStore), AdminGroupRequired())

	ur := &usersRoute{userDAO}
	// list users
	r.GET("/users", ur.listUsers)
	// get user by username
	r.GET("/user/:username", ur.getUser)
	// create user
	r.POST("/user", ur.createUser)
	// update user
	r.PUT("/user/:username", ur.updateUser)
	// delete user
	r.DELETE("/user/:username", ur.deleteUser)

	gr := &groupsRoute{userDAO, groupDAO}
	// list groups
	r.GET("/groups", gr.listGroups)
	// get group and it's users
	r.GET("/group/:name", gr.getGroup)
	// create group
	r.POST("/group", gr.createGroup)
	// update group
	r.PUT("/group/:name", gr.updateGroup)
	// delete group
	r.DELETE("/group/:name", gr.deleteGroup)

	dr := &drivesRoute{config, driveDAO, driveDataDAO, rootDrive}
	// get drive factories
	r.GET("/drive-factories", dr.getDriveFactories)
	// get drives
	r.GET("/drives", dr.getDrives)
	// add drive
	r.POST("/drive", dr.createDrive)
	// update drive
	r.PUT("/drive/:name", dr.updateDrive)
	// delete drive
	r.DELETE("/drive/:name", dr.deleteDrive)
	// get drive initialization information
	r.POST("/drive/:name/init-config", dr.getDriveInitConfig)
	// init drive
	r.POST("/drive/:name/init", dr.doDriveInit)
	// reload drives
	r.POST("/drives/reload", dr.reloadDrives)

	cr := &configRoute{access, permissionDAO, pathMetaDAO, pathMountDAO, optionsDAO, rootDrive, bus}
	// get by path
	r.GET("/path-permissions/*path", cr.getPathPermissions)
	// save path permissions
	r.PUT("/path-permissions/*path", cr.savePathPermissions)

	// get all path meta
	r.GET("/path-meta", cr.getAllPathMeta)
	// create or add path meta
	r.POST("/path-meta/*path", cr.savePathMeta)
	// delete path meta by path
	r.DELETE("/path-meta/*path", cr.deletePathMeta)

	// save options
	r.PUT("/options", cr.saveOptions)
	// get option
	r.GET("/options/:keys", cr.getOptions)

	// save mounts
	r.POST("/mount/*to", cr.savePathMounts)

	mr := &miscRoute{access, permissionDAO, pathMountDAO, rootDrive, search, ch}
	// index files
	r.POST("/search/index/*path", mr.updateSearcherIndexes)
	// clean all PathPermission and PathMount that is point to invalid path
	r.POST("/clean-permissions-mounts", mr.cleanupInvalidPathPermissionsAndMounts)
	// get service stats
	r.GET("/stats", mr.getSystemStats)
	// clean drive cache
	r.DELETE("/drive-cache/:name", mr.clearDriveCache)

	// region script drives

	scriptDriveRoutesGroup := r.Group("/scripts")
	sdr := &scriptDrivesRoute{config: config}
	// get available drives from repository
	scriptDriveRoutesGroup.GET("/available", sdr.getAvailableDrives)
	// get installed drives
	scriptDriveRoutesGroup.GET("/installed", sdr.getInstalledDrives)
	// install drive
	scriptDriveRoutesGroup.POST("/install/:name", sdr.installDrive)
	// uninstall drive
	scriptDriveRoutesGroup.DELETE("/uninstall/:name", sdr.uninstallDrive)
	// get drive script content
	scriptDriveRoutesGroup.GET("/content/:name", sdr.getDriveScriptContent)
	// update drive script content
	scriptDriveRoutesGroup.PUT("/content/:name", sdr.saveDriveScriptContent)

	jobsRoutesGroup := r.Group("/jobs")
	jr := &jobsRoute{ch, runner, jobExecutor, scheduledDAO}
	// get all job definitions
	jobsRoutesGroup.GET("/definitions", jr.getJobsDefinitions)
	// get all created jobs
	jobsRoutesGroup.GET("", jr.getJobs)
	// create job
	jobsRoutesGroup.POST("", jr.createJob)
	// update job
	jobsRoutesGroup.PUT("/:id", jr.updateJob)
	// delete job
	jobsRoutesGroup.DELETE("/:id", jr.deleteJob)
	// get all executions
	jobsRoutesGroup.GET("/executions", jr.getAllExecutions)
	// execute a job
	jobsRoutesGroup.POST("/execution", jr.executeJob)
	// cancel job execution
	jobsRoutesGroup.PUT("/execution/:id/cancel", jr.cancelJobExecution)
	// delete job execution
	jobsRoutesGroup.DELETE("/execution/:id", jr.deleteJobExecution)
	// delete job executions by jobId
	jobsRoutesGroup.DELETE("/execution", jr.deleteJobExecutionsByJobId)
	// execute job script code
	jobsRoutesGroup.POST("/script-eval", jr.scriptEval)

	fbr := &fileBucketConfigRoute{fileBucketDAO}
	// get all file buckets
	r.GET("/file-buckets", fbr.getAllBuckets)
	// create file bucket
	r.POST("/file-bucket", fbr.createBucket)
	// update file bucket
	r.PUT("/file-bucket/:name", fbr.updateBucket)
	// delete file bucket
	r.DELETE("/file-bucket/:name", fbr.deleteBucket)

	return nil
}
