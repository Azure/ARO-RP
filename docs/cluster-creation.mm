<map version="1.0.1">
<!-- To view this file, download free mind mapping software FreeMind from http://freemind.sourceforge.net -->
<node CREATED="1590501647125" ID="ID_1180463267" MODIFIED="1590501731010" TEXT="Cluster creation">
<node CREATED="1590502294517" ID="ID_1734178417" MODIFIED="1590502299641" POSITION="right" TEXT="pre-steps">
<node CREATED="1590502253749" ID="ID_797314692" MODIFIED="1590502264066" TEXT="customer optionally gets Red Hat pull secret"/>
<node CREATED="1590501802453" ID="ID_904198521" MODIFIED="1590501917964" TEXT="register ARO RP (az provider register)"/>
<node CREATED="1590501904633" ID="ID_1079717712" MODIFIED="1590501908156" TEXT="create vnets"/>
<node CREATED="1590501926018" ID="ID_1086806568" MODIFIED="1590501929862" TEXT="create resource group"/>
</node>
<node CREATED="1590501919601" ID="ID_161177817" MODIFIED="1591025651949" POSITION="right" TEXT="az aro create (python/az/aro/azext_aro/custom.py)">
<node CREATED="1590502276404" ID="ID_929220713" MODIFIED="1591025695487" TEXT="static parameter validation"/>
<node CREATED="1591025679701" ID="ID_101192528" MODIFIED="1591025686655" TEXT="validate RP is registered to subscription"/>
<node CREATED="1591025696331" ID="ID_619046778" MODIFIED="1591025702207" TEXT="dynamic parameter validation"/>
<node CREATED="1590502123499" ID="ID_698894632" MODIFIED="1590502126726" TEXT="cluster application is created"/>
<node CREATED="1590502140300" ID="ID_2890021" MODIFIED="1590502146177" TEXT="cluster service principal is created"/>
<node CREATED="1590502114963" ID="ID_1250417331" MODIFIED="1590502152495" TEXT="role assignments are created for cluster SP and RP SP"/>
<node CREATED="1590501932641" ID="ID_855800116" MODIFIED="1591026874493" TEXT="PUT to RP including cluster SP and wait for HTTP 201 response"/>
<node CREATED="1590502081826" ID="ID_265795688" MODIFIED="1591026893807" TEXT="az polls asynchronous operation record to wait for RP completion"/>
<node CREATED="1591026894337" ID="ID_482750948" MODIFIED="1591026912237" TEXT="output last received openShiftClusters document state"/>
</node>
<node CREATED="1590501950473" ID="ID_975936412" MODIFIED="1590501955854" POSITION="right" TEXT="PUT goes through ARM routing"/>
<node CREATED="1590501957577" ID="ID_1034488002" MODIFIED="1591025982923" POSITION="right" TEXT="PUT is handled by RP frontend (pkg/frontend)">
<node CREATED="1591026038303" ID="ID_1841508673" MODIFIED="1591026123524" TEXT="HTTP request traverses middleware functions (pkg/frontend/middleware)">
<node CREATED="1591026063872" ID="ID_681924085" MODIFIED="1591026067131" TEXT="Lowercase"/>
<node CREATED="1591026072536" ID="ID_331440489" MODIFIED="1591026073715" TEXT="Log"/>
<node CREATED="1591026076192" ID="ID_211604015" MODIFIED="1591026077308" TEXT="Metrics"/>
<node CREATED="1591026077520" ID="ID_902396769" MODIFIED="1591026082020" TEXT="Panic"/>
<node CREATED="1591026082272" ID="ID_875080718" MODIFIED="1591026083467" TEXT="Headers"/>
<node CREATED="1591026083688" ID="ID_736348554" MODIFIED="1591026215254" TEXT="Validate (URL parameters)"/>
<node CREATED="1591026091720" ID="ID_853456946" MODIFIED="1591026092788" TEXT="Body"/>
<node CREATED="1591026102200" ID="ID_381029164" MODIFIED="1591026312142" TEXT="Authenticated (not for /healthz)"/>
</node>
<node CREATED="1591026323778" ID="ID_1682226583" MODIFIED="1591026382359" TEXT="putOrPatchOpenShiftCluster route (pkg/frontend/openshiftcluster_putorpatch.go)">
<node CREATED="1591026411900" ID="ID_1709517415" MODIFIED="1591026417240" TEXT="validate subscription state"/>
<node CREATED="1591026487468" ID="ID_1239570824" MODIFIED="1591026490857" TEXT="unmarshal request body"/>
<node CREATED="1590502373094" FOLDED="true" ID="ID_326312598" MODIFIED="1591026486162" TEXT="(static) validation (request body)">
<node CREATED="1591026267090" ID="ID_277833875" MODIFIED="1591026267090" TEXT=""/>
</node>
<node CREATED="1590502328534" ID="ID_715119661" MODIFIED="1591026615514" TEXT="set ProvisioningState to &quot;Creating&quot; to queue to backend"/>
<node CREATED="1591026511765" ID="ID_692144509" MODIFIED="1591026516609" TEXT="allocate monitoring bucket"/>
<node CREATED="1591026556253" ID="ID_1483905360" MODIFIED="1591026579930" TEXT="create asynchronous operation record which client can poll on"/>
<node CREATED="1590502173652" ID="ID_1206384235" MODIFIED="1590502186327" TEXT="cluster record created in CosmosDB"/>
<node CREATED="1591026646790" ID="ID_782853728" MODIFIED="1591026664562" TEXT="return cluster record to end user (excluding secret fields)"/>
</node>
</node>
<node CREATED="1590502343830" ID="ID_1390328428" MODIFIED="1590502349073" POSITION="right" TEXT="Creation is handled by RP backend">
<node CREATED="1590502362021" ID="ID_705716060" MODIFIED="1590502381721" TEXT="dynamic validation"/>
<node CREATED="1590503945624" ID="ID_364348246" MODIFIED="1590503974245" TEXT="cluster ACR token created, calculate cluster pull secret"/>
<node CREATED="1590502779514" ID="ID_1648402988" MODIFIED="1590502787787" TEXT="generate installconfig">
<node CREATED="1590503163495" ID="ID_1467487990" MODIFIED="1590503170131" TEXT="includes references to vnets"/>
<node CREATED="1590503923288" ID="ID_716993735" MODIFIED="1590503934619" TEXT="includes cluster pull secret"/>
<node CREATED="1590504630936" ID="ID_1227083335" MODIFIED="1590504638020" TEXT="includes cluster service principal"/>
<node CREATED="1590504969578" ID="ID_347262687" MODIFIED="1590504975958" TEXT="includes AZ information"/>
</node>
<node CREATED="1590502975629" ID="ID_349047649" MODIFIED="1590502991682" TEXT="phase 1: bootstrap">
<node CREATED="1590502696186" ID="ID_45495378" MODIFIED="1590502706845" TEXT="register cluster dns record if appropriate"/>
<node CREATED="1590502019483" ID="ID_960981413" MODIFIED="1590504508210" TEXT="installation graph is generated from installconfig using the vendored installer">
<node CREATED="1590504187746" ID="ID_552125362" MODIFIED="1590504995151" TEXT="includes 1 or 3 worker machineset(s)"/>
<node CREATED="1590504643520" ID="ID_412100703" MODIFIED="1590504650678" TEXT="includes secret with cluster service principal"/>
<node CREATED="1590502642273" ID="ID_1960649964" MODIFIED="1590502655973" TEXT="includes bootstrap ignition config"/>
<node CREATED="1590503832727" ID="ID_337657349" MODIFIED="1590503838619" TEXT="includes bootstrap cluster assets"/>
</node>
<node CREATED="1590502512176" ID="ID_1871853681" MODIFIED="1590502625805" TEXT="create initial cluster resources using ARM (not terraform)">
<node CREATED="1590502944932" ID="ID_1463280690" MODIFIED="1590502954136" TEXT="network security group(s?)"/>
<node CREATED="1590503299824" ID="ID_1815786954" MODIFIED="1590503313188" TEXT="&quot;cluster&quot; storage account for ignition config"/>
<node CREATED="1590503353899" ID="ID_842297870" MODIFIED="1590503358909" TEXT="storage containers"/>
</node>
<node CREATED="1590503186392" ID="ID_1357558604" MODIFIED="1590503199515" TEXT="attach nsgs to subnets"/>
<node CREATED="1590503330393" ID="ID_446093690" MODIFIED="1590503369030" TEXT="write ignition config into blob in &quot;cluster&quot; storage account"/>
<node CREATED="1590502466183" ID="ID_986457782" MODIFIED="1590502472499" TEXT="billing record is created"/>
<node CREATED="1590502914276" ID="ID_1293565635" MODIFIED="1590502919544" TEXT="create more cluster resources using ARM">
<node CREATED="1590502687977" ID="ID_365124781" MODIFIED="1590502692037" TEXT="cluster private dns"/>
<node CREATED="1590502560440" ID="ID_355611943" MODIFIED="1590502562684" TEXT="bootstrap VM"/>
<node CREATED="1590502572168" ID="ID_1424819241" MODIFIED="1590502577596" TEXT="3 master VMs"/>
<node CREATED="1590503419666" ID="ID_832407609" MODIFIED="1590503426741" TEXT="load balancers"/>
<node CREATED="1590504421422" ID="ID_74218882" MODIFIED="1590504424993" TEXT="private link service"/>
</node>
<node CREATED="1590504438350" ID="ID_1774929298" MODIFIED="1590504456657" TEXT="create private endpoint in RP resource group and connect PE/PLS"/>
<node CREATED="1590502724282" ID="ID_991825848" MODIFIED="1590502740270" TEXT="create cluster signed TLS certificates if appropriate"/>
<node CREATED="1590503510899" ID="ID_309529824" MODIFIED="1590505140320" TEXT="wait for bootstrap completion configmap"/>
<node CREATED="1590503470139" ID="ID_1195352875" MODIFIED="1590503477543" TEXT="install mdsd"/>
</node>
<node CREATED="1590503666981" ID="ID_818219301" MODIFIED="1590504857645" TEXT="bootstrap VM executes (bootkube.sh)">
<node CREATED="1590504788473" ID="ID_1189966395" MODIFIED="1590504799942" TEXT="bootstrap vm provides initial apiserver implementation behind ILB"/>
<node CREATED="1590503732622" ID="ID_1984986404" MODIFIED="1590504725261" TEXT="bootstrap etcd and wait for stability"/>
<node CREATED="1590503858838" ID="ID_1075791103" MODIFIED="1590503901019" TEXT="runs release payload image"/>
<node CREATED="1590503876391" ID="ID_1346442592" MODIFIED="1590503896395" TEXT="more cluster assets are generated"/>
<node CREATED="1590504219587" ID="ID_18180478" MODIFIED="1590504287943" TEXT="assets repeatedly applies assets against running cluster"/>
<node CREATED="1590505112901" ID="ID_1690485668" MODIFIED="1590505123264" TEXT="writes a bootstrap completed configmap"/>
</node>
<node CREATED="1590504291004" ID="ID_1622034039" MODIFIED="1590504299947" TEXT="gradually, cluster operators start">
<node CREATED="1590503769591" ID="ID_1573177316" MODIFIED="1590503796731" TEXT="machine-api operator starts running">
<node CREATED="1590503671133" ID="ID_617037512" MODIFIED="1590503807631" TEXT="worker VMs created"/>
</node>
<node CREATED="1590504342445" ID="ID_362731312" MODIFIED="1590504354176" TEXT="ingress dependent on worker VMs"/>
<node CREATED="1590504315508" ID="ID_1589775695" MODIFIED="1590504341528" TEXT="console starts running (dependent on ingress)"/>
</node>
<node CREATED="1590502992446" ID="ID_833769786" MODIFIED="1590502997130" TEXT="phase 2: post-bootstrap">
<node CREATED="1590502868427" ID="ID_724737877" MODIFIED="1590502880919" TEXT="delete bootstrap VM, nic, disk"/>
<node CREATED="1590505072660" ID="ID_1726041669" MODIFIED="1590505104472" TEXT="delete ignition config (unencrypted), but keep the graph (encrypted)"/>
<node CREATED="1590503075141" ID="ID_835163685" MODIFIED="1590503083905" TEXT="apply signed TLS certificates to cluster"/>
<node CREATED="1590503236984" ID="ID_1257617294" MODIFIED="1590503249309" TEXT="disable cluster Cincinnati configuration"/>
<node CREATED="1590503389626" ID="ID_762508405" MODIFIED="1590503398533" TEXT="update console branding"/>
<node CREATED="1590504072049" ID="ID_1216289806" MODIFIED="1590504087702" TEXT="alertmanager configuration step (hack)"/>
</node>
<node CREATED="1590503612499" ID="ID_17207191" MODIFIED="1590503627720" TEXT="cluster record provisioningState -&gt; &quot;Succeeded&quot; or &quot;Failed&quot;"/>
</node>
<node CREATED="1590504014890" ID="ID_1199416675" MODIFIED="1590504022476" POSITION="right" TEXT="Not part of cluster creation">
<node CREATED="1590504023473" ID="ID_1023037898" MODIFIED="1590504034228" TEXT="No cluster-specific Geneva configuration"/>
</node>
<node CREATED="1590504543431" ID="ID_294285508" MODIFIED="1590504550763" POSITION="right" TEXT="Potential follow-up actions">
<node CREATED="1590504551615" ID="ID_398187299" MODIFIED="1590504574803" TEXT="Document data flow `az aro create` parameters -&gt; where they are actually used"/>
<node CREATED="1590504835345" ID="ID_1657913982" MODIFIED="1590504841453" TEXT="Go into more detail on bootstrap VM steps"/>
<node CREATED="1591025935206" ID="ID_1810135696" MODIFIED="1591025958914" TEXT="Re-review ARM manifest (RP registration)"/>
<node CREATED="1591026798041" ID="ID_961966630" MODIFIED="1591026806252" TEXT="Look at how to represent one process waiting on another"/>
</node>
</node>
</map>
