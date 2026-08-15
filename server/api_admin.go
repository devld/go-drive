package server

import (
	"go-drive/common"
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/server/job"
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
	jobExecutor *job.JobExecutor,
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
	jobDAO *storage.JobDAO,
	fileBucketDAO *storage.FileBucketDAO) error {

	r = r.Group("/admin", TokenAuth(tokenStore), AdminGroupRequired())

	ur := &usersRoute{userDAO}
	// list users
	r.GET("/users", ur.listUsers)
	// get user by username
	r.GET("/users/:username", ur.getUser)
	// create user
	r.POST("/users", ur.createUser)
	// update user
	r.PUT("/users/:username", ur.updateUser)
	// delete user
	r.DELETE("/users/:username", ur.deleteUser)

	gr := &groupsRoute{groupDAO}
	// list groups
	r.GET("/groups", gr.listGroups)
	// get group and it's users
	r.GET("/groups/:name", gr.getGroup)
	// create group
	r.POST("/groups", gr.createGroup)
	// update group
	r.PUT("/groups/:name", gr.updateGroup)
	// delete group
	r.DELETE("/groups/:name", gr.deleteGroup)

	dr := &drivesRoute{config, driveDAO, driveDataDAO, rootDrive}
	// get drive factories
	r.GET("/drive-factories", dr.getDriveFactories)
	// get drives
	r.GET("/drives", dr.getDrives)
	// add drive
	r.POST("/drives", dr.createDrive)
	// update drive
	r.PUT("/drives/:name", dr.updateDrive)
	// delete drive
	r.DELETE("/drives/:name", dr.deleteDrive)
	// get drive initialization information
	r.POST("/drives/:name/init-config", dr.getDriveInitConfig)
	// init drive
	r.POST("/drives/:name/init", dr.doDriveInit)
	// reload drives
	r.POST("/drives/reload", dr.reloadDrives)

	cr := &configRoute{
		access:        access,
		permissionDAO: permissionDAO,
		pathMetaDAO:   pathMetaDAO,
		pathMountDAO:  pathMountDAO,
		optionsDAO:    optionsDAO,
		rootDrive:     rootDrive,
		bus:           bus,
	}
	// get by path
	r.GET("/path-permissions", cr.getPathPermissions)
	// save path permissions
	r.PUT("/path-permissions", cr.savePathPermissions)

	// get all path metadata
	r.GET("/path-metadata", cr.getAllPathMeta)
	// create or update path metadata
	r.PUT("/path-metadata", cr.savePathMeta)
	// delete path metadata by path
	r.DELETE("/path-metadata", cr.deletePathMeta)

	// save options
	r.PUT("/options", cr.saveOptions)
	// get option
	r.GET("/options/:keys", cr.getOptions)

	// save mounts
	r.PUT("/path-mounts", cr.savePathMounts)

	mr := &miscRoute{access, permissionDAO, pathMountDAO, rootDrive, search, ch}
	// index files
	r.PUT("/search-indexes", mr.updateSearcherIndexes)
	// clean all PathPermission and PathMount that is point to invalid path
	r.POST("/maintenance/path-rules/cleanup", mr.cleanupInvalidPathPermissionsAndMounts)
	// get service stats
	r.GET("/stats", mr.getSystemStats)
	// clean drive cache
	r.DELETE("/drives/:name/cache", mr.clearDriveCache)

	// region script drives

	scriptDriveRoutesGroup := r.Group("/drive-scripts")
	sdr := &scriptDrivesRoute{config: config, runner: runner}
	// list repository and installed drive scripts
	scriptDriveRoutesGroup.GET("", sdr.listDriveScripts)
	// sync available drives from repository
	scriptDriveRoutesGroup.POST("/sync", sdr.syncAvailableDrives)
	// install drive
	scriptDriveRoutesGroup.PUT("/:name", sdr.installDrive)
	// uninstall drive
	scriptDriveRoutesGroup.DELETE("/:name", sdr.uninstallDrive)
	// get drive script content
	scriptDriveRoutesGroup.GET("/:name/content", sdr.getDriveScriptContent)
	// update drive script content
	scriptDriveRoutesGroup.PUT("/:name/content", sdr.saveDriveScriptContent)

	jobsRoutesGroup := r.Group("/jobs")
	jr := &jobsRoute{ch, runner, jobExecutor, jobDAO}
	// get all job definitions
	r.GET("/job-definitions", jr.getJobsDefinitions)
	// get all created jobs
	jobsRoutesGroup.GET("", jr.getJobs)
	// create job
	jobsRoutesGroup.POST("", jr.createJob)
	// update job
	jobsRoutesGroup.PUT("/:id", jr.updateJob)
	// delete job
	jobsRoutesGroup.DELETE("/:id", jr.deleteJob)
	// get all executions
	r.GET("/job-executions", jr.getAllExecutions)
	// execute a job
	r.POST("/job-executions", jr.executeJob)
	// cancel job execution
	r.POST("/job-executions/:id/cancel", jr.cancelJobExecution)
	// delete job execution
	r.DELETE("/job-executions/:id", jr.deleteJobExecution)
	// delete job executions by jobId
	r.DELETE("/job-executions", jr.deleteJobExecutionsByJobId)
	// execute job script code
	r.POST("/job-script-evaluations", jr.scriptEval)

	fbr := &fileBucketConfigRoute{fileBucketDAO}
	// get all file buckets
	r.GET("/file-buckets", fbr.getAllBuckets)
	// create file bucket
	r.POST("/file-buckets", fbr.createBucket)
	// update file bucket
	r.PUT("/file-buckets/:name", fbr.updateBucket)
	// delete file bucket
	r.DELETE("/file-buckets/:name", fbr.deleteBucket)

	return nil
}
