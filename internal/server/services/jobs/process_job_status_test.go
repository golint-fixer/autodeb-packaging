package jobs_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"salsa.debian.org/autodeb-team/autodeb/internal/server/models"
	jobsServicePkg "salsa.debian.org/autodeb-team/autodeb/internal/server/services/jobs"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/services/servicestest"
)

func TestProcessJobStatusArchiveUpgrade(t *testing.T) {
	servicesTest := servicestest.SetupTest(t)
	jobsService := servicesTest.Services.Jobs()

	// Create an ArchiveUpgrade
	archiveUpgrade, err := jobsService.CreateArchiveUpgrade(0, 0)
	assert.NoError(t, err)

	// It should have created a SetupArchiveUpgrade Job
	jobs, err := jobsService.GetAllJobsByArchiveUpgradeID(archiveUpgrade.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(jobs), "there should only be one job associated to the archive upgrade")

	setupJob := jobs[0]
	assert.Equal(t, models.JobTypeSetupArchiveUpgrade, setupJob.Type)
	assert.Equal(t, models.JobParentTypeArchiveUpgrade, setupJob.ParentType)
	assert.Equal(t, archiveUpgrade.ID, setupJob.ParentID)

	err = jobsService.ProcessJobStatus(setupJob.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// Create a PackageUpgrade job and mark it as a success
	packageUpgradeJob, err := jobsService.CreatePackageUpgradeJob(archiveUpgrade.ID, "test")
	assert.NoError(t, err)
	err = jobsService.ProcessJobStatus(packageUpgradeJob.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// It should have created an autopkgtest job
	jobs, err = jobsService.GetAllJobsByArchiveUpgradeID(archiveUpgrade.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(jobs), fmt.Sprintf("jobs: %+v", jobs))

}

func TestProcessJobStatusUpgradeAutopkgtest(t *testing.T) {
	servicesTest := servicestest.SetupTest(t)
	jobsService := servicesTest.Services.Jobs()

	archiveUpgrade, err := jobsService.CreateArchiveUpgrade(1, 33)
	assert.NoError(t, err)

	// Create a package upgrade job in the context of an archive upgrade
	upgradeJob, err := jobsService.CreateJob(
		models.JobTypePackageUpgrade, "", 0, models.JobParentTypeArchiveUpgrade, archiveUpgrade.ID,
	)
	assert.NoError(t, err)
	assert.NotNil(t, upgradeJob)

	// There should be two jobs associated with the archive upgrade:
	//  - the upgrade init
	//  - the package upgrade job
	jobs, err := jobsService.GetAllJobsByArchiveUpgradeID(archiveUpgrade.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(jobs))

	// Mark the job as successfull
	err = jobsService.ProcessJobStatus(upgradeJob.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// There should now be a new autopkgtest job associated with the archive upgrade
	jobs, err = jobsService.GetAllJobsByArchiveUpgradeID(archiveUpgrade.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(jobs))

	autopkgTestJob := jobs[2]
	assert.Equal(t, models.JobTypeAutopkgtest, autopkgTestJob.Type)
	assert.Equal(t, models.JobParentTypeArchiveUpgrade, autopkgTestJob.ParentType)
	assert.Equal(t, archiveUpgrade.ID, autopkgTestJob.ParentID)
	assert.Equal(t, upgradeJob.ID, autopkgTestJob.BuildJobID, "this job's build job id should be the package upgrade job")

	// Mark the autopkgtest job as completed
	err = jobsService.ProcessJobStatus(autopkgTestJob.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// There should now be 5 jobs:
	// - the upgrade init
	// - the package upgrade job
	// - the autopkgtest job
	// - main respository job
	// - archive upgrade repository job
	jobs, err = jobsService.GetAllJobsByArchiveUpgradeID(archiveUpgrade.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(jobs))

	archiveUpgradeRepoJob := jobs[3]
	assert.Equal(t, archiveUpgrade.RepositoryName(), archiveUpgradeRepoJob.Input)

	mainRepoJob := jobs[4]
	assert.Equal(t, jobsServicePkg.MainUpgradeRepositoryName, mainRepoJob.Input)
}

func TestProcessJobStatusUploadBuildAndDontForward(t *testing.T) {
	servicesTest := servicestest.SetupTest(t)
	jobsService := servicesTest.Services.Jobs()

	// Create an upload
	upload, err := servicesTest.DB.CreateUpload(22, "testsource", "testversion", "testmaintainer", "testchangedby", false, true)
	assert.NoError(t, err)
	assert.NotNil(t, upload)

	// Create a build job for this upload
	job, err := jobsService.CreateBuildUploadJob(upload.ID)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	// There should be only one job associated with the upload
	jobs, err := jobsService.GetAllJobsByUploadID(upload.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(jobs))

	// Mark the job as failed
	err = jobsService.ProcessJobStatus(job.ID, models.JobStatusFailed)
	assert.NoError(t, err)

	// There should now be no one additional forward job associated with the upload
	jobs, err = jobsService.GetAllJobsByUploadID(upload.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(jobs))
}

func TestProcessJobStatusUploadBuildAndAutopkgTestAndForward(t *testing.T) {
	servicesTest := servicestest.SetupTest(t)
	jobsService := servicesTest.Services.Jobs()

	// Create an upload
	upload, err := servicesTest.DB.CreateUpload(22, "testsource", "testversion", "testmaintainer", "testchangedby", true, true)
	assert.NoError(t, err)
	assert.NotNil(t, upload)

	// Create a build job for this upload
	job, err := jobsService.CreateBuildUploadJob(upload.ID)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	// There should be only one job associated with the upload
	jobs, err := jobsService.GetAllJobsByUploadID(upload.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(jobs))

	// Mark the job as successfull
	err = jobsService.ProcessJobStatus(job.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// There should now be a new autopkgtest job associated with the upload
	jobs, err = jobsService.GetAllJobsByUploadID(upload.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(jobs))

	autopkgTestJob := jobs[1]
	assert.Equal(t, models.JobTypeAutopkgtest, autopkgTestJob.Type)
	assert.Equal(t, upload.ID, autopkgTestJob.ParentID)
	assert.Equal(t, job.ID, autopkgTestJob.BuildJobID, "this job's BuildJobID should be the build job")

	// Mark the autopkgtest job as completed
	err = jobsService.ProcessJobStatus(autopkgTestJob.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// There should now be a forward job associated with the upload
	jobs, err = jobsService.GetAllJobsByUploadID(upload.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(jobs))

	forwardJob := jobs[2]
	assert.Equal(t, models.JobTypeForwardUpload, forwardJob.Type)

	// Mark the forward job as completed
	err = jobsService.ProcessJobStatus(forwardJob.ID, models.JobStatusSuccess)
	assert.NoError(t, err)

	// There should be no additional jobs associated with the upload
	jobs, err = jobsService.GetAllJobsByUploadID(upload.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(jobs))
}