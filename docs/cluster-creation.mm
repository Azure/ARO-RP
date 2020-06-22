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
<node CREATED="1590502373094" ID="ID_326312598" MODIFIED="1591226310312" TEXT="(static) validation (request body)"/>
<node CREATED="1590502328534" ID="ID_715119661" MODIFIED="1591026615514" TEXT="set ProvisioningState to &quot;Creating&quot; to queue to backend"/>
<node CREATED="1591026511765" ID="ID_692144509" MODIFIED="1591026516609" TEXT="allocate monitoring bucket"/>
<node CREATED="1591026556253" ID="ID_1483905360" MODIFIED="1591026579930" TEXT="create asynchronous operation record which client can poll on"/>
<node CREATED="1590502173652" ID="ID_1206384235" MODIFIED="1590502186327" TEXT="cluster record created in CosmosDB"/>
<node CREATED="1591026646790" ID="ID_782853728" MODIFIED="1591026664562" TEXT="return cluster record to end user (excluding secret fields)"/>
</node>
</node>
<node CREATED="1590502343830" ID="ID_1390328428" MODIFIED="1591226374839" POSITION="right" TEXT="Creation is handled by RP backend (pkg/backend)">
<node CREATED="1591226420068" ID="ID_1572125659" MODIFIED="1591226749539" TEXT="backends race to dequeue cluster record, one backend wins and takes the lease"/>
<node CREATED="1591226456124" ID="ID_1974865526" MODIFIED="1591226692283" TEXT="heartbeat process starts, updating lease (prevents other backends dequeueing the same record)"/>
<node CREATED="1590502362021" ID="ID_705716060" MODIFIED="1591226556577" TEXT="dynamic validation (pkg/api/validate/openshiftcluster_validatedynamic.go)">
<node CREATED="1591226558510" ID="ID_1625452736" MODIFIED="1591226574889" TEXT="vnet permissions"/>
<node CREATED="1591226575270" ID="ID_942065036" MODIFIED="1591226578729" TEXT="route table permissions"/>
<node CREATED="1591226579238" ID="ID_101213817" MODIFIED="1591226585105" TEXT="vnet validation">
<node CREATED="1591226592406" ID="ID_785623303" MODIFIED="1591226597577" TEXT="overlapping subnets"/>
<node CREATED="1591226598909" ID="ID_501781834" MODIFIED="1591226604569" TEXT="dns servers misconfigured"/>
</node>
<node CREATED="1591226606166" ID="ID_967308470" MODIFIED="1591226613066" TEXT="necessary resource providers registered"/>
<node CREATED="1591226613422" ID="ID_1357148973" MODIFIED="1591226620890" TEXT="sufficient quota"/>
</node>
<node CREATED="1591226960499" ID="ID_1868560504" MODIFIED="1591226965054" TEXT="prepare data for installconfig">
<node CREATED="1590503945624" ID="ID_364348246" MODIFIED="1591226860333" TEXT="create ACR token and password"/>
<node CREATED="1591226841609" ID="ID_1989042463" MODIFIED="1591226850525" TEXT="generate cluster ssh key, storage suffix"/>
<node CREATED="1591226864585" ID="ID_1180722902" MODIFIED="1591226870934" TEXT="calculate cluster pull secret">
<node CREATED="1591226879450" ID="ID_554370730" MODIFIED="1591226889989" TEXT="merge user&apos;s pull secret with ACR token"/>
<node CREATED="1591226892993" ID="ID_1864768217" MODIFIED="1591226921013" TEXT="remove cloud.openshift.com to prevent telemetry data leaving Azure"/>
</node>
<node CREATED="1591226933922" ID="ID_1151216440" MODIFIED="1591226949445" TEXT="calculate availability zone configuration for masters and workers"/>
</node>
<node CREATED="1590502779514" ID="ID_1648402988" MODIFIED="1591227053975" TEXT="populate installconfig struct">
<node CREATED="1590503163495" ID="ID_1467487990" MODIFIED="1590503170131" TEXT="includes references to vnets"/>
<node CREATED="1590503923288" ID="ID_716993735" MODIFIED="1590503934619" TEXT="includes cluster pull secret"/>
<node CREATED="1590504630936" ID="ID_1227083335" MODIFIED="1590504638020" TEXT="includes cluster service principal"/>
<node CREATED="1590504969578" ID="ID_347262687" MODIFIED="1590504975958" TEXT="includes AZ information"/>
<node CREATED="1591226998618" ID="ID_1805181178" MODIFIED="1591227013926" TEXT="includes RHCOS image co-ordinates"/>
</node>
<node CREATED="1591227019507" ID="ID_877475648" MODIFIED="1591227022798" TEXT="validate installconfig"/>
<node CREATED="1591227024803" ID="ID_1706136491" MODIFIED="1591227099863" TEXT="run install (pkg/install/install.go)">
<node CREATED="1590502975629" ID="ID_349047649" MODIFIED="1590502991682" TEXT="phase 1: bootstrap">
<node CREATED="1590502696186" ID="ID_45495378" MODIFIED="1591228716914" TEXT="register external cluster dns record if appropriate"/>
<node CREATED="1591227143660" ID="ID_846465412" MODIFIED="1591227147513" TEXT="deploy storage template">
<node CREATED="1591227191973" ID="ID_926546882" MODIFIED="1591227236506" TEXT="generate ClusterID install graph node"/>
<node CREATED="1591227202861" ID="ID_7835241" MODIFIED="1591227271713" TEXT="create cluster resource group"/>
<node CREATED="1591227305518" ID="ID_1281381149" MODIFIED="1591227319210" TEXT="retrieve service principal ID for cluster application"/>
<node CREATED="1591227616113" ID="ID_99542609" MODIFIED="1591227620690" TEXT="build ARM template">
<node CREATED="1591227378351" ID="ID_863743809" MODIFIED="1591227392331" TEXT="role assignment (CSP -&gt; Contributor)"/>
<node CREATED="1590503353899" ID="ID_842297870" MODIFIED="1590503358909" TEXT="storage containers">
<node CREATED="1591227420599" ID="ID_1974776406" MODIFIED="1591227520348" TEXT="ignition - serves bootstrap ignition config"/>
<node CREATED="1591227422095" ID="ID_599514623" MODIFIED="1591227534052" TEXT="aro - saves encrypted copy of the install graph"/>
</node>
<node CREATED="1590502944932" ID="ID_1463280690" MODIFIED="1591227441620" TEXT="network security groups">
<node CREATED="1591227442696" ID="ID_832435734" MODIFIED="1591227445579" TEXT="control plane"/>
<node CREATED="1591227445791" ID="ID_1013416500" MODIFIED="1591227475388" TEXT="worker nodes"/>
</node>
<node CREATED="1591227555473" ID="ID_1235869362" MODIFIED="1591227604165" TEXT="deny assignment for anyone but RP and CSP"/>
</node>
<node CREATED="1590502512176" ID="ID_1871853681" MODIFIED="1591228512592" TEXT="deploy ARM template and wait"/>
<node CREATED="1590502019483" ID="ID_960981413" MODIFIED="1591227674230" TEXT="generate install graph from installconfig using the vendored installer">
<node CREATED="1590504187746" ID="ID_552125362" MODIFIED="1590504995151" TEXT="includes 1 or 3 worker machineset(s)"/>
<node CREATED="1590504643520" ID="ID_412100703" MODIFIED="1590504650678" TEXT="includes secret with cluster service principal"/>
<node CREATED="1590502642273" ID="ID_1960649964" MODIFIED="1590502655973" TEXT="includes bootstrap ignition config"/>
<node CREATED="1590503832727" ID="ID_337657349" MODIFIED="1590503838619" TEXT="includes bootstrap cluster assets"/>
</node>
<node CREATED="1591227657818" ID="ID_150264491" MODIFIED="1591227666030" TEXT="write install graph to storage container"/>
</node>
<node CREATED="1590503186392" ID="ID_1357558604" MODIFIED="1590503199515" TEXT="attach nsgs to subnets"/>
<node CREATED="1590502466183" ID="ID_986457782" MODIFIED="1591227804528" TEXT="create billing record"/>
<node CREATED="1591228496836" ID="ID_48294449" MODIFIED="1591228521314" TEXT="create more cluster resources using ARM">
<node CREATED="1590502914276" ID="ID_1293565635" MODIFIED="1591228494903" TEXT="build ARM template">
<node CREATED="1590502687977" ID="ID_365124781" MODIFIED="1591227844880" TEXT="cluster private dns zone and recordsets">
<node CREATED="1591227861044" ID="ID_425280122" MODIFIED="1591227902481" TEXT="A api-int (ILB)"/>
<node CREATED="1591227892957" ID="ID_441536775" MODIFIED="1591227904929" TEXT="A api (LB)"/>
<node CREATED="1591227865156" ID="ID_299076322" MODIFIED="1591227921802" TEXT="A etcd-{0,1,2}"/>
<node CREATED="1591227922045" ID="ID_1443092651" MODIFIED="1591227928113" TEXT="SRV _etcd-server-ssl._tcp"/>
</node>
<node CREATED="1591227946885" ID="ID_1851897858" MODIFIED="1591227955690" TEXT="virtual network link (joins dns zone to vnet)"/>
<node CREATED="1590504421422" ID="ID_74218882" MODIFIED="1590504424993" TEXT="private link service"/>
<node CREATED="1591227976710" ID="ID_534503069" MODIFIED="1591228384910" TEXT="public IP for API server inbound (-pip-v4)"/>
<node CREATED="1590503419666" ID="ID_832407609" MODIFIED="1591228413239" TEXT="public load balancer for API server">
<node CREATED="1591228019279" ID="ID_705458648" MODIFIED="1591228024938" TEXT="configuration depends on private/public apiserver"/>
<node CREATED="1591228060791" ID="ID_965676248" MODIFIED="1591228065298" TEXT=":6443"/>
</node>
<node CREATED="1591228013534" ID="ID_456510197" MODIFIED="1591228416703" TEXT="internal load balancer for API server">
<node CREATED="1591228067071" ID="ID_1368782287" MODIFIED="1591228069571" TEXT=":6443"/>
<node CREATED="1591228070631" ID="ID_1886145277" MODIFIED="1591228092291" TEXT=":22623 (serves ignition)"/>
</node>
<node CREATED="1590502560440" ID="ID_355611943" MODIFIED="1591228080059" TEXT="bootstrap NIC, VM">
<node CREATED="1591228154568" ID="ID_1645425455" MODIFIED="1591228562617" TEXT="customdata points to ignition config in storage container (uses SAS token)"/>
<node CREATED="1591228234385" ID="ID_1732972442" MODIFIED="1591228235844" TEXT="diagnostics profile saves serial output"/>
</node>
<node CREATED="1590502572168" ID="ID_1424819241" MODIFIED="1591228084131" TEXT="3 master NICs and VMs">
<node CREATED="1591228175697" ID="ID_1866133243" MODIFIED="1591228185700" TEXT="customdata points to internal load balancer :22623"/>
<node CREATED="1591228220865" ID="ID_828339793" MODIFIED="1591228231245" TEXT="diagnostics profile saves serial output"/>
</node>
<node CREATED="1591228243945" ID="ID_813134746" MODIFIED="1591228475528" TEXT="public IP for worker nodes outbound (-outbound-pip-v4)"/>
<node CREATED="1591228407499" ID="ID_1099527640" MODIFIED="1591228471559" TEXT="public load balancer for worker nodes"/>
</node>
<node CREATED="1591228541789" ID="ID_867243099" MODIFIED="1591228576721" TEXT="generate SAS token and pass as ARM template parameter"/>
<node CREATED="1591228501468" ID="ID_1952696774" MODIFIED="1591228509160" TEXT="deploy ARM template and wait"/>
</node>
<node CREATED="1590504438350" ID="ID_1774929298" MODIFIED="1590504456657" TEXT="create private endpoint in RP resource group and connect PE/PLS"/>
<node CREATED="1591228610285" ID="ID_283255789" MODIFIED="1591228635082" TEXT="update API server IP">
<node CREATED="1591228635941" ID="ID_1512066616" MODIFIED="1591228722858" TEXT="update external cluster dns record which was registered at start (if applicable)"/>
<node CREATED="1591228724615" ID="ID_296105763" MODIFIED="1591228737658" TEXT="store private endpoint IP in database"/>
</node>
<node CREATED="1590502724282" ID="ID_991825848" MODIFIED="1591228752850" TEXT="create signed TLS certificates if appropriate">
<node CREATED="1591228754151" ID="ID_1438424488" MODIFIED="1591228757187" TEXT="API server"/>
<node CREATED="1591228757447" ID="ID_837500356" MODIFIED="1591228762675" TEXT="ingress"/>
</node>
<node CREATED="1591228786871" ID="ID_781093330" MODIFIED="1591228790899" TEXT="(create kubernetes clients)"/>
<node CREATED="1590503510899" ID="ID_309529824" MODIFIED="1590505140320" TEXT="wait for bootstrap completion configmap"/>
<node CREATED="1590503470139" ID="ID_1195352875" MODIFIED="1591228834699" TEXT="install geneva logging (mdsd)"/>
<node CREATED="1591228836448" ID="ID_1513120256" MODIFIED="1591228848228" TEXT="install ifreload kernel bug workaround"/>
<node CREATED="1591228852080" ID="ID_78262830" MODIFIED="1591228855821" TEXT="proceed to phase 2"/>
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
<node CREATED="1590502992446" ID="ID_833769786" MODIFIED="1591227133568" TEXT="phase 2: remove-bootstrap">
<node CREATED="1590502868427" ID="ID_724737877" MODIFIED="1590502880919" TEXT="delete bootstrap VM, nic, disk"/>
<node CREATED="1590505072660" ID="ID_1726041669" MODIFIED="1590505104472" TEXT="delete ignition config (unencrypted), but keep the graph (encrypted)"/>
<node CREATED="1590503075141" ID="ID_835163685" MODIFIED="1590503083905" TEXT="apply signed TLS certificates to cluster"/>
<node CREATED="1590503236984" ID="ID_1257617294" MODIFIED="1590503249309" TEXT="disable cluster Cincinnati configuration"/>
<node CREATED="1590503389626" ID="ID_762508405" MODIFIED="1590503398533" TEXT="update console branding"/>
<node CREATED="1590504072049" ID="ID_1216289806" MODIFIED="1590504087702" TEXT="alertmanager configuration step (hack)"/>
</node>
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
