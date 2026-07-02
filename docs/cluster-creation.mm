<map version="1.0.1">
<!-- To view this file, download free mind mapping software FreeMind from http://freemind.sourceforge.net -->
<node CREATED="1590501647125" ID="ID_1180463267" MODIFIED="1742320800000" TEXT="Cluster creation">
<node CREATED="1742320931000" ID="ID_100000131" MODIFIED="1742320931000" POSITION="right" TEXT="Two credential models">
<node CREATED="1742320932000" ID="ID_100000132" MODIFIED="1742320932000" TEXT="Service Principal (SP) - legacy">
<node CREATED="1742320933000" ID="ID_100000133" MODIFIED="1742320933000" TEXT="CLI creates AAD application and service principal"/>
<node CREATED="1742320934000" ID="ID_100000134" MODIFIED="1742320934000" TEXT="Role assignments for cluster SP and RP SP"/>
<node CREATED="1742320935000" ID="ID_100000135" MODIFIED="1742320935000" TEXT="Backend uses fixupClusterSPObjectID"/>
</node>
<node CREATED="1742320936000" ID="ID_100000136" MODIFIED="1742320936000" TEXT="Managed Identity (MSI) - new">
<node CREATED="1742320937000" ID="ID_100000137" MODIFIED="1742320937000" TEXT="Customer creates user-assigned managed identity pre-step"/>
<node CREATED="1742320938000" ID="ID_100000138" MODIFIED="1742320938000" TEXT="Customer creates platform workload identities for operators pre-step"/>
<node CREATED="1742320939000" ID="ID_100000139" MODIFIED="1742320939000" TEXT="CLI sets UserAssigned identity and PlatformWorkloadIdentityProfile"/>
<node CREATED="1742320940000" ID="ID_100000140" MODIFIED="1742320940000" TEXT="Frontend stores identity URL and tenant ID from MSI headers"/>
<node CREATED="1742320941000" ID="ID_100000141" MODIFIED="1742320941000" TEXT="Backend generates MSI certificate and creates MSI-authenticated clients"/>
<node CREATED="1742320942000" ID="ID_100000142" MODIFIED="1742320942000" TEXT="Backend creates OIDC federated identity credentials"/>
</node>
</node>
<node CREATED="1742320943000" ID="ID_100000143" MODIFIED="1742320943000" POSITION="right" TEXT="Two install paths">
<node CREATED="1742320944000" ID="ID_100000144" MODIFIED="1742320944000" TEXT="Hive - ClusterDeployment on management cluster">
<node CREATED="1742320945000" ID="ID_100000145" MODIFIED="1742320945000" TEXT="Creates namespace and ClusterDeployment in Hive"/>
<node CREATED="1742320946000" ID="ID_100000146" MODIFIED="1742320946000" TEXT="Waits up to 60 minutes for completion"/>
</node>
<node CREATED="1742320947000" ID="ID_100000147" MODIFIED="1742320947000" TEXT="Podman - local container install">
<node CREATED="1742320948000" ID="ID_100000148" MODIFIED="1742320948000" TEXT="Runs installer in local container"/>
<node CREATED="1742320949000" ID="ID_100000149" MODIFIED="1742320949000" TEXT="Can optionally adopt into Hive after install"/>
</node>
</node>
<node CREATED="1590502294517" ID="ID_1734178417" MODIFIED="1742320800000" POSITION="right" TEXT="pre-steps">
<node CREATED="1590502253749" ID="ID_797314692" MODIFIED="1590502264066" TEXT="customer optionally gets Red Hat pull secret"/>
<node CREATED="1590501802453" ID="ID_904198521" MODIFIED="1590501917964" TEXT="register ARO RP (az provider register)"/>
<node CREATED="1590501904633" ID="ID_1079717712" MODIFIED="1590501908156" TEXT="create vnets"/>
<node CREATED="1590501926018" ID="ID_1086806568" MODIFIED="1590501929862" TEXT="create resource group"/>
<node CREATED="1742320801000" ID="ID_100000001" MODIFIED="1742320801000" TEXT="(MSI path) create user-assigned managed identity"/>
<node CREATED="1742320802000" ID="ID_100000002" MODIFIED="1742320802000" TEXT="(MSI path) create platform workload identities for operators"/>
</node>
<node CREATED="1590501919601" ID="ID_161177817" MODIFIED="1742320800000" POSITION="right" TEXT="az aro create (python/az/aro/azext_aro/custom.py)">
<node CREATED="1590502276404" ID="ID_929220713" MODIFIED="1591025695487" TEXT="static parameter validation"/>
<node CREATED="1591025679701" ID="ID_101192528" MODIFIED="1591025686655" TEXT="validate RP is registered to subscription"/>
<node CREATED="1742320803000" ID="ID_100000003" MODIFIED="1742320803000" TEXT="validate subnets"/>
<node CREATED="1742320804000" ID="ID_100000004" MODIFIED="1742320804000" TEXT="dynamic validation">
<node CREATED="1742320805000" ID="ID_100000005" MODIFIED="1742320805000" TEXT="virtual network permissions"/>
<node CREATED="1742320806000" ID="ID_100000006" MODIFIED="1742320806000" TEXT="resource provider permissions"/>
<node CREATED="1742320807000" ID="ID_100000007" MODIFIED="1742320807000" TEXT="quota validation"/>
<node CREATED="1742320808000" ID="ID_100000008" MODIFIED="1742320808000" TEXT="disk encryption set validation"/>
<node CREATED="1742320809000" ID="ID_100000009" MODIFIED="1742320809000" TEXT="domain validation"/>
<node CREATED="1742320810000" ID="ID_100000010" MODIFIED="1742320810000" TEXT="CIDR range validation"/>
<node CREATED="1742320811000" ID="ID_100000011" MODIFIED="1742320811000" TEXT="version validation"/>
<node CREATED="1742320812000" ID="ID_100000012" MODIFIED="1742320812000" TEXT="outbound type validation (LB vs UserDefinedRouting)"/>
<node CREATED="1742320813000" ID="ID_100000013" MODIFIED="1742320813000" TEXT="(MSI path) managed identity and platform workload identity validation"/>
</node>
<node CREATED="1742320814000" ID="ID_100000014" MODIFIED="1742320814000" TEXT="identity setup (two paths)">
<node CREATED="1590502123499" ID="ID_698894632" MODIFIED="1742320814000" TEXT="(SP path) cluster AAD application is created if client_id not provided"/>
<node CREATED="1590502140300" ID="ID_2890021" MODIFIED="1742320814000" TEXT="(SP path) cluster service principal is created or retrieved"/>
<node CREATED="1742320815000" ID="ID_100000015" MODIFIED="1742320815000" TEXT="(MSI path) set cluster identity to UserAssigned with mi_user_assigned"/>
<node CREATED="1742320816000" ID="ID_100000016" MODIFIED="1742320816000" TEXT="(MSI path) set PlatformWorkloadIdentityProfile with operator identity mappings"/>
</node>
<node CREATED="1742320817000" ID="ID_100000017" MODIFIED="1742320817000" TEXT="build OpenShiftCluster object">
<node CREATED="1742320818000" ID="ID_100000018" MODIFIED="1742320818000" TEXT="ClusterProfile (pull secret, domain, version, FIPS, resource group)"/>
<node CREATED="1742320819000" ID="ID_100000019" MODIFIED="1742320819000" TEXT="NetworkProfile (CIDR ranges, NSG mode, load balancer config, outbound type)"/>
<node CREATED="1742320820000" ID="ID_100000020" MODIFIED="1742320820000" TEXT="MasterProfile and WorkerProfile (VM sizes, encryption at host, subnets)"/>
<node CREATED="1742320821000" ID="ID_100000021" MODIFIED="1742320821000" TEXT="APIServerProfile and IngressProfile (visibility settings)"/>
</node>
<node CREATED="1590502114963" ID="ID_1250417331" MODIFIED="1742320800000" TEXT="ensure_resource_permissions (role assignments for SP/MSI and RP)"/>
<node CREATED="1590501932641" ID="ID_855800116" MODIFIED="1742320800000" TEXT="PUT to RP and wait for HTTP 201 response"/>
<node CREATED="1590502081826" ID="ID_265795688" MODIFIED="1591026893807" TEXT="az polls asynchronous operation record to wait for RP completion"/>
<node CREATED="1591026894337" ID="ID_482750948" MODIFIED="1591026912237" TEXT="output last received openShiftClusters document state"/>
</node>
<node CREATED="1590501950473" ID="ID_975936412" MODIFIED="1590501955854" POSITION="right" TEXT="PUT goes through ARM routing"/>
<node CREATED="1590501957577" ID="ID_1034488002" MODIFIED="1742320800000" POSITION="right" TEXT="PUT is handled by RP frontend (pkg/frontend)">
<node CREATED="1591026038303" ID="ID_1841508673" MODIFIED="1742320800000" TEXT="HTTP request traverses middleware functions (pkg/frontend/middleware)">
<node CREATED="1591026063872" ID="ID_681924085" MODIFIED="1591026067131" TEXT="Lowercase"/>
<node CREATED="1591026072536" ID="ID_331440489" MODIFIED="1591026073715" TEXT="Log"/>
<node CREATED="1591026076192" ID="ID_211604015" MODIFIED="1591026077308" TEXT="Metrics"/>
<node CREATED="1591026077520" ID="ID_902396769" MODIFIED="1591026082020" TEXT="Panic"/>
<node CREATED="1591026082272" ID="ID_875080718" MODIFIED="1591026083467" TEXT="Headers"/>
<node CREATED="1591026083688" ID="ID_736348554" MODIFIED="1591026215254" TEXT="Validate (URL parameters)"/>
<node CREATED="1591026091720" ID="ID_853456946" MODIFIED="1591026092788" TEXT="Body"/>
<node CREATED="1742320822000" ID="ID_100000022" MODIFIED="1742320822000" TEXT="SystemData (ARM system metadata enrichment)"/>
<node CREATED="1591026102200" ID="ID_381029164" MODIFIED="1742320800000" TEXT="Authenticated (MISE or mutual TLS, not for /healthz)"/>
</node>
<node CREATED="1591026323778" ID="ID_1682226583" MODIFIED="1742320800000" TEXT="putOrPatchOpenShiftCluster route (pkg/frontend/openshiftcluster_putorpatch.go)">
<node CREATED="1591026411900" ID="ID_1709517415" MODIFIED="1591026417240" TEXT="validate subscription state"/>
<node CREATED="1591026487468" ID="ID_1239570824" MODIFIED="1591026490857" TEXT="unmarshal request body"/>
<node CREATED="1590502373094" ID="ID_326312598" MODIFIED="1742320800000" TEXT="validation on create (ValidateNewCluster)">
<node CREATED="1742320824000" ID="ID_100000024" MODIFIED="1742320824000" TEXT="staticValidator.Static() - API model validation"/>
<node CREATED="1742320825000" ID="ID_100000025" MODIFIED="1742320825000" TEXT="skuValidator.ValidateVMSku() - VM size availability"/>
<node CREATED="1742320826000" ID="ID_100000026" MODIFIED="1742320826000" TEXT="quotaValidator.ValidateQuota() - compute quota"/>
<node CREATED="1742320827000" ID="ID_100000027" MODIFIED="1742320827000" TEXT="providersValidator.ValidateProviders() - resource provider registration"/>
</node>
<node CREATED="1742320828000" ID="ID_100000028" MODIFIED="1742320828000" TEXT="(MSI path) validatePlatformWorkloadIdentities"/>
<node CREATED="1742320829000" ID="ID_100000029" MODIFIED="1742320829000" TEXT="validate install version is supported"/>
<node CREATED="1590502328534" ID="ID_715119661" MODIFIED="1742320800000" TEXT="set ProvisioningState to Creating to queue to backend"/>
<node CREATED="1742320830000" ID="ID_100000030" MODIFIED="1742320830000" TEXT="allocate monitoring bucket"/>
<node CREATED="1742320831000" ID="ID_100000031" MODIFIED="1742320831000" TEXT="(MSI path) store identity URL and tenant ID in cluster doc"/>
<node CREATED="1742320832000" ID="ID_100000032" MODIFIED="1742320832000" TEXT="set defaults and default operator flags"/>
<node CREATED="1591026556253" ID="ID_1483905360" MODIFIED="1591026579930" TEXT="create asynchronous operation record in CosmosDB which client can poll on"/>
<node CREATED="1590502173652" ID="ID_1206384235" MODIFIED="1590502186327" TEXT="cluster record created in CosmosDB with non-terminal ProvisioningState"/>
<node CREATED="1591026646790" ID="ID_782853728" MODIFIED="1742320800000" TEXT="return cluster record to end user (excluding secrets and sensitive fields)"/>
</node>
</node>
<node CREATED="1590502343830" ID="ID_1390328428" MODIFIED="1742320800000" POSITION="right" TEXT="Creation is handled by RP backend after dequeuing from CosmosDB (pkg/backend)">
<node CREATED="1591226420068" ID="ID_1572125659" MODIFIED="1591226749539" TEXT="backends race to dequeue cluster record, one backend wins and takes the lease"/>
<node CREATED="1591226456124" ID="ID_1974865526" MODIFIED="1591226692283" TEXT="heartbeat process starts, updating lease (prevents other backends dequeueing the same record)"/>
<node CREATED="1742320833000" ID="ID_100000033" MODIFIED="1742320833000" TEXT="load subscription document for RBAC context"/>
<node CREATED="1742320834000" ID="ID_100000034" MODIFIED="1742320834000" TEXT="determine Hive install mode (installViaHive, adoptViaHive, or neither)"/>
<node CREATED="1742320835000" ID="ID_100000035" MODIFIED="1742320835000" TEXT="create cluster manager with database, encryption, billing, Hive, metrics"/>
<node CREATED="1742320836000" ID="ID_100000036" MODIFIED="1742320836000" TEXT="call m.Install(ctx) which runs multi-phase install"/>
</node>
<node CREATED="1742320837000" ID="ID_100000037" MODIFIED="1742320837000" POSITION="right" TEXT="Phase 1: Bootstrap (pkg/cluster/install.go)">
<node CREATED="1742320838000" ID="ID_100000038" MODIFIED="1742320838000" TEXT="credential initialization and validation">
<node CREATED="1742320844000" ID="ID_100000044" MODIFIED="1742320844000" TEXT="(MSI path) ensureClusterMsiCertificate - generate MSI certificate"/>
<node CREATED="1742320845000" ID="ID_100000045" MODIFIED="1742320845000" TEXT="(MSI path) initializeClusterMsiClients - create MSI-authenticated clients"/>
<node CREATED="1742320847000" ID="ID_100000047" MODIFIED="1742320847000" TEXT="(MSI path) platformWorkloadIdentityIDs - populate operator identity IDs"/>
<node CREATED="1742320950000" ID="ID_100000150" MODIFIED="1742320950000" TEXT="validateResources"/>
<node CREATED="1742320951000" ID="ID_100000151" MODIFIED="1742320951000" TEXT="validateZones"/>
<node CREATED="1742320846000" ID="ID_100000046" MODIFIED="1742320846000" TEXT="(MSI path) clusterIdentityIDs - populate cluster MSI object IDs"/>
<node CREATED="1742320848000" ID="ID_100000048" MODIFIED="1742320848000" TEXT="(MSI path) persistPlatformWorkloadIdentityIDs - store IDs in document"/>
<node CREATED="1742320952000" ID="ID_100000152" MODIFIED="1742320952000" TEXT="(SP path) initializeClusterSPClients"/>
<node CREATED="1742320849000" ID="ID_100000049" MODIFIED="1742320849000" TEXT="(SP path) clusterSPObjectID - populate SP object IDs"/>
</node>
<node CREATED="1742320851000" ID="ID_100000051" MODIFIED="1742320851000" TEXT="Azure infrastructure setup">
<node CREATED="1742320857000" ID="ID_100000057" MODIFIED="1742320857000" TEXT="ensurePreconfiguredNSG - if BYO NSG enabled"/>
<node CREATED="1590503945624" ID="ID_364348246" MODIFIED="1742320800000" TEXT="ensureACRToken - ACR credential setup"/>
<node CREATED="1742320858000" ID="ID_100000058" MODIFIED="1742320858000" TEXT="ensureInfraID - generate cluster infrastructure ID"/>
<node CREATED="1742320859000" ID="ID_100000059" MODIFIED="1742320859000" TEXT="ensureSSHKey - generate SSH key pair"/>
<node CREATED="1742320860000" ID="ID_100000060" MODIFIED="1742320860000" TEXT="ensureStorageSuffix"/>
<node CREATED="1742320861000" ID="ID_100000061" MODIFIED="1742320861000" TEXT="populateMTUSize - network MTU sizing"/>
<node CREATED="1742320863000" ID="ID_100000063" MODIFIED="1742320863000" TEXT="createDNS - DNS zone setup"/>
<node CREATED="1742320864000" ID="ID_100000064" MODIFIED="1742320864000" TEXT="createOIDC - OIDC provider creation"/>
<node CREATED="1591227202861" ID="ID_7835241" MODIFIED="1742320800000" TEXT="ensureResourceGroup - create cluster resource group"/>
<node CREATED="1742320853000" ID="ID_100000053" MODIFIED="1742320853000" TEXT="ensureServiceEndpoints - VNet service endpoints"/>
</node>
<node CREATED="1742320865000" ID="ID_100000065" MODIFIED="1742320865000" TEXT="network and compute deployment">
<node CREATED="1742320866000" ID="ID_100000066" MODIFIED="1742320866000" TEXT="setMasterSubnetPolicies"/>
<node CREATED="1742320867000" ID="ID_100000067" MODIFIED="1742320867000" TEXT="deployBaseResourceTemplate - deploy base ARM/bicep (networks, VMs, LBs, NSGs, storage)">
<node CREATED="1590502687977" ID="ID_365124781" MODIFIED="1742320800000" TEXT="cluster private DNS zone and recordsets (api-int, api, etcd, SRV)"/>
<node CREATED="1591227946885" ID="ID_1851897858" MODIFIED="1591227955690" TEXT="virtual network link (joins DNS zone to vnet)"/>
<node CREATED="1590504421422" ID="ID_74218882" MODIFIED="1590504424993" TEXT="private link service"/>
<node CREATED="1742320868000" ID="ID_100000068" MODIFIED="1742320868000" TEXT="public and internal load balancers for API server (:6443, :22623)"/>
<node CREATED="1742320869000" ID="ID_100000069" MODIFIED="1742320869000" TEXT="public IP for API server and worker outbound"/>
<node CREATED="1590502560440" ID="ID_355611943" MODIFIED="1742320800000" TEXT="bootstrap NIC and VM (customdata points to ignition in storage)"/>
<node CREATED="1590502572168" ID="ID_1424819241" MODIFIED="1742320800000" TEXT="3 master NICs and VMs (customdata points to ILB :22623)"/>
<node CREATED="1742320870000" ID="ID_100000070" MODIFIED="1742320870000" TEXT="public load balancer for worker nodes outbound"/>
<node CREATED="1590502944932" ID="ID_1463280690" MODIFIED="1742320800000" TEXT="network security groups (control plane, workers)"/>
<node CREATED="1590503353899" ID="ID_842297870" MODIFIED="1742320800000" TEXT="storage containers (ignition, aro graph)"/>
<node CREATED="1742320871000" ID="ID_100000071" MODIFIED="1742320871000" TEXT="deny assignment for anyone but RP and cluster identity"/>
</node>
<node CREATED="1742320872000" ID="ID_100000072" MODIFIED="1742320872000" TEXT="(MSI path) federateIdentityCredentials - bind Azure identities to k8s service accounts via OIDC"/>
<node CREATED="1742320873000" ID="ID_100000073" MODIFIED="1742320873000" TEXT="attachNSGs - attach NSGs to subnets"/>
</node>
<node CREATED="1742320874000" ID="ID_100000074" MODIFIED="1742320874000" TEXT="API server and networking">
<node CREATED="1742320875000" ID="ID_100000075" MODIFIED="1742320875000" TEXT="updateAPIIPEarly - populate API server IP"/>
<node CREATED="1742320876000" ID="ID_100000076" MODIFIED="1742320876000" TEXT="createOrUpdateRouterIPEarly - set router IP"/>
<node CREATED="1742320877000" ID="ID_100000077" MODIFIED="1742320877000" TEXT="ensureGatewayCreate - create gateway resources"/>
<node CREATED="1590504438350" ID="ID_1774929298" MODIFIED="1742320800000" TEXT="createAPIServerPrivateEndpoint - create PE in RP resource group and connect PE/PLS"/>
</node>
<node CREATED="1590502724282" ID="ID_991825848" MODIFIED="1742320800000" TEXT="createCertificates - TLS certificates (API server, ingress)"/>
<node CREATED="1742320878000" ID="ID_100000078" MODIFIED="1742320878000" TEXT="cluster installation (two paths)">
<node CREATED="1744646500000" ID="ID_200000001" MODIFIED="1744646500000" TEXT="(if installViaHive or adoptViaHive) hiveCreateNamespace - create namespace in management cluster"/>
<node CREATED="1742320879000" ID="ID_100000079" MODIFIED="1742320879000" TEXT="(Hive path) installViaHive">
<node CREATED="1742320881000" ID="ID_100000081" MODIFIED="1742320881000" TEXT="runHiveInstaller - create Hive ClusterDeployment"/>
<node CREATED="1742320882000" ID="ID_100000082" MODIFIED="1742320882000" TEXT="hiveClusterInstallationComplete - wait up to 60 minutes"/>
<node CREATED="1742320883000" ID="ID_100000083" MODIFIED="1742320883000" TEXT="generateKubeconfigs"/>
</node>
<node CREATED="1742320884000" ID="ID_100000084" MODIFIED="1742320884000" TEXT="(Podman path) runPodmanInstaller">
<node CREATED="1742320885000" ID="ID_100000085" MODIFIED="1742320885000" TEXT="runPodmanInstaller - local container-based install"/>
<node CREATED="1742320886000" ID="ID_100000086" MODIFIED="1742320886000" TEXT="generateKubeconfigs"/>
<node CREATED="1742320887000" ID="ID_100000087" MODIFIED="1742320887000" TEXT="(if adoptViaHive) hiveEnsureResources and hiveClusterDeploymentReady (5min)"/>
</node>
<node CREATED="1744646501000" ID="ID_200000002" MODIFIED="1744646501000" TEXT="(if installViaHive or adoptViaHive) hiveResetCorrelationData - reset correlation data before post-install bootstrap"/>
</node>
<node CREATED="1590502466183" ID="ID_986457782" MODIFIED="1742320800000" TEXT="ensureBillingRecord - create/update billing DB record"/>
<node CREATED="1742320888000" ID="ID_100000088" MODIFIED="1742320888000" TEXT="post-install bootstrap phase">
<node CREATED="1742320889000" ID="ID_100000089" MODIFIED="1742320889000" TEXT="initializeKubernetesClients - cluster now running"/>
<node CREATED="1742320890000" ID="ID_100000090" MODIFIED="1742320890000" TEXT="initializeOperatorDeployer"/>
<node CREATED="1742320891000" ID="ID_100000091" MODIFIED="1742320891000" TEXT="apiServersReady - wait up to 30 minutes"/>
<node CREATED="1742320892000" ID="ID_100000092" MODIFIED="1742320892000" TEXT="installAROOperator - deploy ARO operator"/>
<node CREATED="1742320893000" ID="ID_100000093" MODIFIED="1742320893000" TEXT="enableOperatorReconciliation"/>
<node CREATED="1742320894000" ID="ID_100000094" MODIFIED="1742320894000" TEXT="incrInstallPhase - transition to Phase 2"/>
</node>
</node>
<node CREATED="1590503666981" ID="ID_818219301" MODIFIED="1742320800000" POSITION="right" TEXT="bootstrap VM executes (bootkube.sh) - runs in parallel with RP steps">
<node CREATED="1590504788473" ID="ID_1189966395" MODIFIED="1590504799942" TEXT="bootstrap VM provides initial apiserver implementation behind ILB"/>
<node CREATED="1590503732622" ID="ID_1984986404" MODIFIED="1590504725261" TEXT="bootstrap etcd and wait for stability"/>
<node CREATED="1590503858838" ID="ID_1075791103" MODIFIED="1590503901019" TEXT="runs release payload image"/>
<node CREATED="1590503876391" ID="ID_1346442592" MODIFIED="1590503896395" TEXT="more cluster assets are generated"/>
<node CREATED="1590504219587" ID="ID_18180478" MODIFIED="1590504287943" TEXT="assets repeatedly applied against running cluster"/>
<node CREATED="1590505112901" ID="ID_1690485668" MODIFIED="1590505123264" TEXT="writes a bootstrap completed configmap"/>
<node CREATED="1590504291004" ID="ID_1622034039" MODIFIED="1590504299947" TEXT="gradually, cluster operators start">
<node CREATED="1590503769591" ID="ID_1573177316" MODIFIED="1590503796731" TEXT="machine-api operator starts running">
<node CREATED="1590503671133" ID="ID_617037512" MODIFIED="1590503807631" TEXT="worker VMs created"/>
</node>
<node CREATED="1590504342445" ID="ID_362731312" MODIFIED="1590504354176" TEXT="ingress dependent on worker VMs"/>
<node CREATED="1590504315508" ID="ID_1589775695" MODIFIED="1590504341528" TEXT="console starts running (dependent on ingress)"/>
</node>
</node>
<node CREATED="1742320895000" ID="ID_100000095" MODIFIED="1742320895000" POSITION="right" TEXT="Phase 2: RemoveBootstrap (pkg/cluster/install.go)">
<node CREATED="1742320896000" ID="ID_100000096" MODIFIED="1742320896000" TEXT="initialize clients">
<node CREATED="1742320897000" ID="ID_100000097" MODIFIED="1742320897000" TEXT="initializeKubernetesClients"/>
<node CREATED="1742320898000" ID="ID_100000098" MODIFIED="1742320898000" TEXT="initializeOperatorDeployer"/>
</node>
<node CREATED="1742320899000" ID="ID_100000099" MODIFIED="1742320899000" TEXT="bootstrap removal">
<node CREATED="1590502868427" ID="ID_724737877" MODIFIED="1742320800000" TEXT="removeBootstrap - delete bootstrap VM, NIC, disk"/>
<node CREATED="1590505072660" ID="ID_1726041669" MODIFIED="1742320800000" TEXT="removeBootstrapIgnition - delete unencrypted ignition config (keep encrypted graph)"/>
</node>
<node CREATED="1742320900000" ID="ID_100000100" MODIFIED="1742320900000" TEXT="API server and node health">
<node CREATED="1742320901000" ID="ID_100000101" MODIFIED="1742320901000" TEXT="apiServersReady (30min timeout)"/>
<node CREATED="1742320902000" ID="ID_100000102" MODIFIED="1742320902000" TEXT="configureAPIServerCertificate - apply signed TLS certificate"/>
<node CREATED="1742320903000" ID="ID_100000103" MODIFIED="1742320903000" TEXT="apiServersReady (30min timeout) - recheck after cert change"/>
<node CREATED="1742320904000" ID="ID_100000104" MODIFIED="1742320904000" TEXT="minimumWorkerNodesReady (30min timeout)"/>
</node>
<node CREATED="1742320905000" ID="ID_100000105" MODIFIED="1742320905000" TEXT="console and UI">
<node CREATED="1742320906000" ID="ID_100000106" MODIFIED="1742320906000" TEXT="operatorConsoleExists (30min timeout)"/>
<node CREATED="1590503389626" ID="ID_762508405" MODIFIED="1742320800000" TEXT="updateConsoleBranding - custom ARO branding"/>
<node CREATED="1742320907000" ID="ID_100000107" MODIFIED="1742320907000" TEXT="operatorConsoleReady (20min timeout)"/>
</node>
<node CREATED="1742320908000" ID="ID_100000108" MODIFIED="1742320908000" TEXT="OpenShift configuration">
<node CREATED="1742320909000" ID="ID_100000109" MODIFIED="1742320909000" TEXT="disableSamples - disable sample operators"/>
<node CREATED="1742320910000" ID="ID_100000110" MODIFIED="1742320910000" TEXT="disableOperatorHubSources"/>
<node CREATED="1590503236984" ID="ID_1257617294" MODIFIED="1742320800000" TEXT="disableUpdates - lock cluster version"/>
<node CREATED="1742320911000" ID="ID_100000111" MODIFIED="1742320911000" TEXT="clusterVersionReady (30min timeout)"/>
</node>
<node CREATED="1742320912000" ID="ID_100000112" MODIFIED="1742320912000" TEXT="ARO operator stabilization">
<node CREATED="1742320913000" ID="ID_100000113" MODIFIED="1742320913000" TEXT="aroDeploymentReady (20min timeout)"/>
<node CREATED="1742320914000" ID="ID_100000114" MODIFIED="1742320914000" TEXT="updateClusterData - update cluster metadata"/>
</node>
<node CREATED="1742320915000" ID="ID_100000115" MODIFIED="1742320915000" TEXT="cluster networking and storage">
<node CREATED="1742320916000" ID="ID_100000116" MODIFIED="1742320916000" TEXT="configureIngressCertificate - ingress TLS certificate"/>
<node CREATED="1742320917000" ID="ID_100000117" MODIFIED="1742320917000" TEXT="ingressControllerReady (30min timeout)"/>
<node CREATED="1742320918000" ID="ID_100000118" MODIFIED="1742320918000" TEXT="configureDefaultStorageClass"/>
<node CREATED="1742320919000" ID="ID_100000119" MODIFIED="1742320919000" TEXT="removeAzureFileCSIStorageClass - cleanup legacy storage"/>
</node>
<node CREATED="1742320920000" ID="ID_100000120" MODIFIED="1742320920000" TEXT="finalization">
<node CREATED="1742320921000" ID="ID_100000121" MODIFIED="1742320921000" TEXT="disableOperatorReconciliation - stop operator changes"/>
<node CREATED="1742320922000" ID="ID_100000122" MODIFIED="1742320922000" TEXT="clusterOperatorsHaveSettled (30min timeout) - verify stability"/>
<node CREATED="1742320923000" ID="ID_100000123" MODIFIED="1742320923000" TEXT="finishInstallation - clear Install field, mark complete"/>
</node>
</node>
<node CREATED="1590503612499" ID="ID_17207191" MODIFIED="1742320800000" POSITION="right" TEXT="cluster record provisioningState set to Succeeded or Failed"/>
</node>
</map>
