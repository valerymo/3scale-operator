package component

import (
	"fmt"

	appsv1 "github.com/openshift/api/apps/v1"
	templatev1 "github.com/openshift/api/template/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Redis struct {
	// TemplateParameters
	// TemplateObjects
	// CLI Flags??? should be in this object???
	options []string
	Options *RedisOptions
}

type RedisOptions struct {
	redisNonRequiredOptions
	redisRequiredOptions
}

type redisRequiredOptions struct {
	appLabel string
	image    string
}

type redisNonRequiredOptions struct {
}

func NewRedis(options []string) *Redis {
	redis := &Redis{
		options: options,
	}
	return redis
}

type RedisOptionsBuilder struct {
	options RedisOptions
}

func (r *RedisOptionsBuilder) AppLabel(appLabel string) {
	r.options.appLabel = appLabel
}

func (r *RedisOptionsBuilder) Image(image string) {
	r.options.image = image
}

func (r *RedisOptionsBuilder) Build() (*RedisOptions, error) {
	err := r.setRequiredOptions()
	if err != nil {
		return nil, err
	}

	r.setNonRequiredOptions()

	return &r.options, nil
}

func (r *RedisOptionsBuilder) setRequiredOptions() error {
	if r.options.appLabel == "" {
		return fmt.Errorf("no AppLabel has been provided")
	}
	if r.options.image == "" {
		return fmt.Errorf("no Redis Image has been provided")
	}

	return nil
}

func (r *RedisOptionsBuilder) setNonRequiredOptions() {

}

type RedisOptionsProvider interface {
	GetRedisOptions() *RedisOptions
}

type CLIRedisOptionsProvider struct {
}

func (o *CLIRedisOptionsProvider) GetRedisOptions() (*RedisOptions, error) {
	rob := RedisOptionsBuilder{}
	rob.AppLabel("${APP_LABEL}")
	rob.Image("${REDIS_IMAGE}")
	res, err := rob.Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create Redis Options - %s", err)
	}
	return res, nil
}

func (redis *Redis) AssembleIntoTemplate(template *templatev1.Template, otherComponents []Component) {
	redis.buildParameters(template)
	redis.addObjectsIntoTemplate(template)
}

func (redis *Redis) GetObjects() ([]runtime.RawExtension, error) {
	objects := redis.buildObjects()
	return objects, nil
}

func (redis *Redis) addObjectsIntoTemplate(template *templatev1.Template) {
	// TODO move this outside this specific method
	optionsProvider := CLIRedisOptionsProvider{}
	redisOpts, err := optionsProvider.GetRedisOptions()
	_ = err
	redis.Options = redisOpts
	objects := redis.buildObjects()
	template.Objects = append(template.Objects, objects...)
}

func (redis *Redis) buildObjects() []runtime.RawExtension {
	backendRedisObjects := redis.buildBackendRedisObjects()
	systemRedisObjects := redis.buildSystemRedisObjects()

	objects := backendRedisObjects
	objects = append(objects, systemRedisObjects...)
	return objects
}

func (redis *Redis) PostProcess(template *templatev1.Template, otherComponents []Component) {

}

func (redis *Redis) buildParameters(template *templatev1.Template) {
	parameters := []templatev1.Parameter{
		{
			Name:        "REDIS_IMAGE",
			Description: "Redis image to use",
			Required:    true,
			Value:       "registry.access.redhat.com/rhscl/redis-32-rhel7:3.2",
		},
	}
	template.Parameters = append(template.Parameters, parameters...)
}

func (redis *Redis) buildBackendRedisObjects() []runtime.RawExtension {
	dc := redis.buildBackendDeploymentConfig()
	bs := redis.buildBackendService()
	cm := redis.buildBackendConfigMap()
	bpvc := redis.buildBackendRedisPVC()
	objects := []runtime.RawExtension{
		runtime.RawExtension{Object: dc},
		runtime.RawExtension{Object: bs},
		runtime.RawExtension{Object: cm},
		runtime.RawExtension{Object: bpvc},
	}
	return objects
}

func (redis *Redis) buildBackendDeploymentConfig() *appsv1.DeploymentConfig {
	return &appsv1.DeploymentConfig{
		TypeMeta:   redis.buildDeploymentConfigTypeMeta(),
		ObjectMeta: redis.buildDeploymentConfigObjectMeta(),
		Spec:       redis.buildDeploymentConfigSpec(),
	}
}

func (redis *Redis) buildDeploymentConfigTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "DeploymentConfig",
		APIVersion: "apps.openshift.io/v1",
	}
}

const ()

func (redis *Redis) buildDeploymentConfigObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   backendRedisObjectMetaName,
		Labels: redis.buildLabelsForDeploymentConfigObjectMeta(),
	}
}

const (
	backendRedisObjectMetaName    = "backend-redis"
	backendRedisDCSelectorName    = backendRedisObjectMetaName
	backendComponentNameLabel     = "backend"
	backendComponentElementLabel  = "redis"
	backendRedisStorageVolumeName = "backend-redis-storage"
	backendRedisConfigVolumeName  = "redis-config"
	backendRedisConfigMapKey      = "redis.conf"
	backendRedisContainerName     = "backend-redis"
	backendRedisContainerCommand  = "/opt/rh/rh-redis32/root/usr/bin/redis-server"
)

func (redis *Redis) buildLabelsForDeploymentConfigObjectMeta() map[string]string {
	return map[string]string{
		"app":                      redis.Options.appLabel,
		"3scale.component":         backendComponentNameLabel,
		"3scale.component-element": backendComponentElementLabel,
	}
}

func (redis *Redis) buildDeploymentConfigSpec() appsv1.DeploymentConfigSpec {
	return appsv1.DeploymentConfigSpec{
		Template: redis.buildPodTemplateSpec(),
		Strategy: redis.buildDeploymentStrategy(),
		Selector: redis.buildDeploymentConfigSelector(),
		Replicas: 1, //TODO make this configurable via flag
		Triggers: redis.buildDeploymentConfigTriggers(),
	}
}

func (redis *Redis) buildDeploymentStrategy() appsv1.DeploymentStrategy {
	return appsv1.DeploymentStrategy{
		Type: appsv1.DeploymentStrategyTypeRecreate,
	}
}

func (redis *Redis) getSelectorLabels() map[string]string {
	return map[string]string{
		"deploymentConfig": backendRedisDCSelectorName,
	}
}

func (redis *Redis) buildDeploymentConfigSelector() map[string]string {
	return redis.getSelectorLabels()
}

func (redis *Redis) buildDeploymentConfigTriggers() appsv1.DeploymentTriggerPolicies {
	return appsv1.DeploymentTriggerPolicies{
		appsv1.DeploymentTriggerPolicy{
			Type: appsv1.DeploymentTriggerOnConfigChange,
		},
	}
}

func (redis *Redis) buildPodTemplateSpec() *v1.PodTemplateSpec {
	return &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			ServiceAccountName: "amp", //TODO make this configurable via flag
			Volumes:            redis.buildPodVolumes(),
			Containers:         redis.buildPodContainers(),
		},
		ObjectMeta: redis.buildPodObjectMeta(),
	}
}

func (redis *Redis) buildPodObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Labels: map[string]string{
			"deploymentConfig":         backendRedisDCSelectorName,
			"app":                      redis.Options.appLabel,
			"3scale.component":         backendComponentNameLabel,
			"3scale.component-element": backendComponentElementLabel,
		},
	}
}

func (redis *Redis) buildPodVolumes() []v1.Volume {
	return []v1.Volume{
		v1.Volume{
			Name: backendRedisStorageVolumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: backendRedisStorageVolumeName,
				},
			},
		},
		v1.Volume{
			Name: backendRedisConfigVolumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{ //The name of the ConfigMap
						Name: backendRedisConfigVolumeName,
					},
					Items: []v1.KeyToPath{
						v1.KeyToPath{
							Key:  backendRedisConfigMapKey,
							Path: backendRedisConfigMapKey,
						},
					},
				},
			},
		},
	}
}

func (redis *Redis) buildPodContainers() []v1.Container {
	return []v1.Container{
		v1.Container{
			Image:           redis.Options.image,
			ImagePullPolicy: v1.PullIfNotPresent,
			Name:            backendRedisContainerName,
			Command:         redis.buildPodContainerCommand(),
			Args:            redis.buildPodContainerCommandArgs(),
			Resources:       redis.buildPodContainerResourceLimits(),
			ReadinessProbe:  redis.buildPodContainerReadinessProbe(),
			LivenessProbe:   redis.buildPodContainerLivenessProbe(),
			VolumeMounts:    redis.buildPodContainerVolumeMounts(),
		},
	}
}

func (redis *Redis) buildPodContainerCommand() []string {
	return []string{
		backendRedisContainerCommand,
	}
}

func (redis *Redis) buildPodContainerCommandArgs() []string {
	return []string{
		"/etc/redis.d/redis.conf",
		"--daemonize",
		"no",
	}
}

func (redis *Redis) buildPodContainerResourceLimits() v1.ResourceRequirements {
	return v1.ResourceRequirements{ //TODO Make this configurable via an option flag.
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2000m"), //another option was to use resource.Parse which would not panic and return an error if error
			v1.ResourceMemory: resource.MustParse("32Gi"),
		},
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1000m"),
			v1.ResourceMemory: resource.MustParse("1024Mi"),
		},
	}
}

func (redis *Redis) buildPodContainerReadinessProbe() *v1.Probe {
	return &v1.Probe{
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{
					"container-entrypoint",
					"bash",
					"-c",
					"redis-cli set liveness-probe \"`date`\" | grep OK",
				},
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
		TimeoutSeconds:      1,
	}
}

func (redis *Redis) buildPodContainerLivenessProbe() *v1.Probe {
	return &v1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		Handler: v1.Handler{
			TCPSocket: &v1.TCPSocketAction{
				Port: intstr.FromInt(6379),
			},
		},
	}
}

func (redis *Redis) buildPodContainerVolumeMounts() []v1.VolumeMount {
	return []v1.VolumeMount{
		v1.VolumeMount{
			Name:      backendRedisStorageVolumeName,
			MountPath: "/var/lib/redis/data",
		},
		v1.VolumeMount{
			Name:      backendRedisConfigVolumeName,
			MountPath: "/etc/redis.d/",
		},
	}
}

func (redis *Redis) buildBackendService() *v1.Service {
	return &v1.Service{
		ObjectMeta: redis.buildServiceObjectMeta(),
		TypeMeta:   redis.buildServiceTypeMeta(),
		Spec:       redis.buildServiceSpec(),
	}
}

func (redis *Redis) buildServiceObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   "backend-redis",
		Labels: redis.buildLabelsForServiceObjectMeta(),
	}
}

func (redis *Redis) buildLabelsForServiceObjectMeta() map[string]string {
	return map[string]string{
		"app":                      redis.Options.appLabel,
		"3scale.component":         "backend",
		"3scale.component-element": "redis",
	}
}

func (redis *Redis) buildServiceTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}
}

func (redis *Redis) buildServiceSpec() v1.ServiceSpec {
	return v1.ServiceSpec{
		Ports:    redis.buildServicePorts(),
		Selector: redis.buildServiceSelector(),
	}
}

func (redis *Redis) buildServicePorts() []v1.ServicePort {
	return []v1.ServicePort{
		v1.ServicePort{
			Port:       6379,
			TargetPort: intstr.FromInt(6379),
			Protocol:   v1.ProtocolTCP,
		},
	}
}

func (redis *Redis) buildServiceSelector() map[string]string {
	return map[string]string{
		"deploymentConfig": backendRedisDCSelectorName,
	}
}

func (redis *Redis) buildBackendConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: redis.buildConfigMapObjectMeta(),
		TypeMeta:   redis.buildConfigMapTypeMeta(),
		Data:       redis.buildConfigMapData(),
	}
}

func (redis *Redis) buildConfigMapObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   backendRedisConfigVolumeName,
		Labels: redis.buildLabelsForConfigMapObjectMeta(),
	}
}

func (redis *Redis) buildLabelsForConfigMapObjectMeta() map[string]string {
	return map[string]string{
		"app":                      redis.Options.appLabel,
		"3scale.component":         "system", // TODO should also be redis???
		"3scale.component-element": "redis",
	}
}

func (redis *Redis) buildConfigMapTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
}

func (redis *Redis) buildConfigMapData() map[string]string {
	return map[string]string{
		"redis.conf": redis.getRedisConfData(),
	}
}

func (redis *Redis) getRedisConfData() string { // TODO read this from a real file
	return `protected-mode no

port 6379

timeout 0
tcp-keepalive 300

daemonize no
supervised no

loglevel notice

databases 16

save 900 1
save 300 10
save 60 10000

stop-writes-on-bgsave-error yes

rdbcompression yes
rdbchecksum yes

dbfilename dump.rdb

slave-serve-stale-data yes
slave-read-only yes

repl-diskless-sync no
repl-disable-tcp-nodelay no

appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
aof-load-truncated yes

lua-time-limit 5000

activerehashing no

aof-rewrite-incremental-fsync yes
dir /var/lib/redis/data
`
}

func (redis *Redis) buildBackendRedisPVC() *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: redis.buildPVCObjectMeta(),
		TypeMeta:   redis.buildPVCTypeMeta(),
		Spec:       redis.buildPVCSpec(),
		// TODO be able to configure StorageClass in case one wants to be used
	}
}

func (redis *Redis) buildPVCObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   backendRedisStorageVolumeName,
		Labels: redis.buildLabelsForServiceObjectMeta(),
	}
}

func (redis *Redis) buildLabelsForPVCObjectMeta() map[string]string {
	return map[string]string{
		"app":                      redis.Options.appLabel,
		"3scale.component":         "backend",
		"3scale.component-element": "redis",
	}
}

func (redis *Redis) buildPVCTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	}
}

func (redis *Redis) buildPVCSpec() v1.PersistentVolumeClaimSpec {
	return v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{
			v1.ReadWriteOnce, // TODO be able to configure this because we have different volume access modes for different claims
		},
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}

////// Begin System Redis

func (redis *Redis) buildSystemRedisObjects() []runtime.RawExtension {

	systemRedisDC := &appsv1.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentConfig",
			APIVersion: "apps.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "system-redis",
			Labels: map[string]string{"3scale.component": "system", "3scale.component-element": "redis", "app": redis.Options.appLabel},
		},
		Spec: appsv1.DeploymentConfigSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyType("Recreate"),
			},
			MinReadySeconds: 0,
			Triggers: appsv1.DeploymentTriggerPolicies{
				appsv1.DeploymentTriggerPolicy{
					Type: appsv1.DeploymentTriggerType("ConfigChange")},
			},
			Replicas: 1,
			Selector: map[string]string{"deploymentConfig": "system-redis"},
			Template: &v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"3scale.component": "system", "3scale.component-element": "redis", "app": redis.Options.appLabel, "deploymentConfig": "system-redis"},
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						v1.Volume{
							Name: "system-redis-storage",
							VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: "system-redis-storage",
								ReadOnly:  false}},
						}, v1.Volume{
							Name: "redis-config",
							VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "redis-config",
								},
								Items: []v1.KeyToPath{
									v1.KeyToPath{
										Key:  "redis.conf",
										Path: "redis.conf"}}}}},
					},
					Containers: []v1.Container{
						v1.Container{
							Name:    "system-redis",
							Image:   redis.Options.image,
							Command: []string{"/opt/rh/rh-redis32/root/usr/bin/redis-server"},
							Args:    []string{"/etc/redis.d/redis.conf", "--daemonize", "no"},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("500m"),
									v1.ResourceMemory: resource.MustParse("32Gi"),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("150m"),
									v1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							VolumeMounts: []v1.VolumeMount{
								v1.VolumeMount{
									Name:      "system-redis-storage",
									ReadOnly:  false,
									MountPath: "/var/lib/redis/data",
								}, v1.VolumeMount{
									Name:      "redis-config",
									ReadOnly:  false,
									MountPath: "/etc/redis.d/"},
							},
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{TCPSocket: &v1.TCPSocketAction{
									Port: intstr.IntOrString{
										Type:   intstr.Type(0),
										IntVal: 6379}},
								},
								InitialDelaySeconds: 10,
								TimeoutSeconds:      0,
								PeriodSeconds:       5,
								SuccessThreshold:    0,
								FailureThreshold:    0,
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									Exec: &v1.ExecAction{
										Command: []string{"container-entrypoint", "bash", "-c", "redis-cli set liveness-probe \"`date`\" | grep OK"}},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    0,
								FailureThreshold:    0,
							},
							TerminationMessagePath: "/dev/termination-log",
							ImagePullPolicy:        v1.PullPolicy("IfNotPresent"),
						},
					},
				}},
		},
	}

	systemRedisPVC := &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-redis-storage",
			Labels: map[string]string{
				"3scale.component":         "system",
				"3scale.component-element": "redis",
				"app":                      redis.Options.appLabel,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.PersistentVolumeAccessMode("ReadWriteOnce"),
			},
			Resources: v1.ResourceRequirements{Requests: v1.ResourceList{"storage": resource.MustParse("1Gi")}}}}

	objects := []runtime.RawExtension{
		runtime.RawExtension{Object: systemRedisDC},
		runtime.RawExtension{Object: systemRedisPVC},
	}

	return objects
}

////// End System Redis