---
applyTo: "internal/nexusapi/*.go"
title: Nexus Repository Manager REST API v3.84.1-01
---

<h1 id="nexus-repository-manager-rest-api">Nexus Repository Manager REST API v3.84.1-01</h1>

> Scroll down for code samples, example requests and responses. Select a language for code samples from the tabs above or the mobile navigation menu.

Base URLs:

* <a href="/service/rest/">/service/rest/</a>

<h1 id="nexus-repository-manager-rest-api-assets">assets</h1>

## getAssets

<a id="opIdgetAssets"></a>

`GET /v1/assets`

*List assets*

<h3 id="getassets-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|continuationToken|query|string|false|A token returned by a prior request. If present, the next page of results are returned|
|repository|query|string|true|Repository from which you would like to retrieve assets.|

> Example responses

> 200 Response

```json
{
  "items": [
    {
      "downloadUrl": "string",
      "path": "string",
      "id": "string",
      "repository": "string",
      "format": "string",
      "checksum": {
        "property1": "string",
        "property2": "string"
      },
      "contentType": "string",
      "lastModified": "2019-08-24T14:15:22Z",
      "lastDownloaded": "2019-08-24T14:15:22Z",
      "uploader": "string",
      "uploaderIp": "string",
      "fileSize": 0,
      "blobCreated": "2019-08-24T14:15:22Z",
      "blobStoreName": "string"
    }
  ],
  "continuationToken": "string"
}
```

<h3 id="getassets-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|[PageAssetXO](#schemapageassetxo)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to list assets|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Parameter 'repository' is required|None|

<aside class="success">
This operation does not require authentication
</aside>

## getAssetById

<a id="opIdgetAssetById"></a>

`GET /v1/assets/{id}`

*Get a single asset*

<h3 id="getassetbyid-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|string|true|Id of the asset to get|

> Example responses

> 200 Response

```json
{
  "downloadUrl": "string",
  "path": "string",
  "id": "string",
  "repository": "string",
  "format": "string",
  "checksum": {
    "property1": "string",
    "property2": "string"
  },
  "contentType": "string",
  "lastModified": "2019-08-24T14:15:22Z",
  "lastDownloaded": "2019-08-24T14:15:22Z",
  "uploader": "string",
  "uploaderIp": "string",
  "fileSize": 0,
  "blobCreated": "2019-08-24T14:15:22Z",
  "blobStoreName": "string"
}
```

<h3 id="getassetbyid-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|[AssetXO](#schemaassetxo)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to get asset|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Asset not found|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Malformed ID|None|

<aside class="success">
This operation does not require authentication
</aside>

## deleteAsset

<a id="opIddeleteAsset"></a>

`DELETE /v1/assets/{id}`

*Delete a single asset*

<h3 id="deleteasset-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|string|true|Id of the asset to delete|

<h3 id="deleteasset-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|Asset was successfully deleted|None|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to delete asset|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Asset not found|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Malformed ID|None|

<aside class="success">
This operation does not require authentication
</aside>

<h1 id="nexus-repository-manager-rest-api-components">components</h1>

## getComponents

<a id="opIdgetComponents"></a>

`GET /v1/components`

*List components*

<h3 id="getcomponents-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|continuationToken|query|string|false|A token returned by a prior request. If present, the next page of results are returned|
|repository|query|string|true|Repository from which you would like to retrieve components|

> Example responses

> 200 Response

```json
{
  "items": [
    {
      "id": "string",
      "repository": "string",
      "format": "string",
      "group": "string",
      "name": "string",
      "version": "string",
      "assets": [
        {
          "downloadUrl": "string",
          "path": "string",
          "id": "string",
          "repository": "string",
          "format": "string",
          "checksum": {
            "property1": "string",
            "property2": "string"
          },
          "contentType": "string",
          "lastModified": "2019-08-24T14:15:22Z",
          "lastDownloaded": "2019-08-24T14:15:22Z",
          "uploader": "string",
          "uploaderIp": "string",
          "fileSize": 0,
          "blobCreated": "2019-08-24T14:15:22Z",
          "blobStoreName": "string"
        }
      ]
    }
  ],
  "continuationToken": "string"
}
```

<h3 id="getcomponents-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|[PageComponentXO](#schemapagecomponentxo)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to list components|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Parameter 'repository' is required|None|

<aside class="success">
This operation does not require authentication
</aside>

## uploadComponent

<a id="opIduploadComponent"></a>

`POST /v1/components`

*Upload a single component*

> Body parameter

```yaml
yum.directory: string
yum.tag: string
yum.asset: string
yum.asset.filename: string
r.tag: string
r.asset: string
r.asset.pathId: string
apt.tag: string
apt.asset: string
pypi.tag: string
pypi.asset: string
maven2.groupId: string
maven2.artifactId: string
maven2.version: string
maven2.generate-pom: true
maven2.packaging: string
maven2.tag: string
maven2.asset1: string
maven2.asset1.classifier: string
maven2.asset1.extension: string
maven2.asset2: string
maven2.asset2.classifier: string
maven2.asset2.extension: string
maven2.asset3: string
maven2.asset3.classifier: string
maven2.asset3.extension: string
raw.directory: string
raw.tag: string
raw.asset1: string
raw.asset1.filename: string
raw.asset2: string
raw.asset2.filename: string
raw.asset3: string
raw.asset3.filename: string
npm.tag: string
npm.asset: string
nuget.tag: string
nuget.asset: string
rubygems.tag: string
rubygems.asset: string
helm.tag: string
helm.asset: string
docker.tag: string
docker.asset: string

```

<h3 id="uploadcomponent-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|repository|query|string|true|Name of the repository to which you would like to upload the component|
|body|body|object|false|none|
|» yum.directory|body|string|false|yum Directory|
|» yum.tag|body|string|false|yum Tag|
|» yum.asset|body|string(binary)|false|yum Asset|
|» yum.asset.filename|body|string|false|yum Asset  Filename|
|» r.tag|body|string|false|r Tag|
|» r.asset|body|string(binary)|false|r Asset|
|» r.asset.pathId|body|string|false|r Asset  Package Path|
|» apt.tag|body|string|false|apt Tag|
|» apt.asset|body|string(binary)|false|apt Asset|
|» pypi.tag|body|string|false|pypi Tag|
|» pypi.asset|body|string(binary)|false|pypi Asset|
|» maven2.groupId|body|string|false|maven2 Group ID|
|» maven2.artifactId|body|string|false|maven2 Artifact ID|
|» maven2.version|body|string|false|maven2 Version|
|» maven2.generate-pom|body|boolean|false|maven2 Generate a POM file with these coordinates|
|» maven2.packaging|body|string|false|maven2 Packaging|
|» maven2.tag|body|string|false|maven2 Tag|
|» maven2.asset1|body|string(binary)|false|maven2 Asset 1|
|» maven2.asset1.classifier|body|string|false|maven2 Asset 1 Classifier|
|» maven2.asset1.extension|body|string|false|maven2 Asset 1 Extension|
|» maven2.asset2|body|string(binary)|false|maven2 Asset 2|
|» maven2.asset2.classifier|body|string|false|maven2 Asset 2 Classifier|
|» maven2.asset2.extension|body|string|false|maven2 Asset 2 Extension|
|» maven2.asset3|body|string(binary)|false|maven2 Asset 3|
|» maven2.asset3.classifier|body|string|false|maven2 Asset 3 Classifier|
|» maven2.asset3.extension|body|string|false|maven2 Asset 3 Extension|
|» raw.directory|body|string|false|raw Directory|
|» raw.tag|body|string|false|raw Tag|
|» raw.asset1|body|string(binary)|false|raw Asset 1|
|» raw.asset1.filename|body|string|false|raw Asset 1 Filename|
|» raw.asset2|body|string(binary)|false|raw Asset 2|
|» raw.asset2.filename|body|string|false|raw Asset 2 Filename|
|» raw.asset3|body|string(binary)|false|raw Asset 3|
|» raw.asset3.filename|body|string|false|raw Asset 3 Filename|
|» npm.tag|body|string|false|npm Tag|
|» npm.asset|body|string(binary)|false|npm Asset|
|» nuget.tag|body|string|false|nuget Tag|
|» nuget.asset|body|string(binary)|false|nuget Asset|
|» rubygems.tag|body|string|false|rubygems Tag|
|» rubygems.asset|body|string(binary)|false|rubygems Asset|
|» helm.tag|body|string|false|helm Tag|
|» helm.asset|body|string(binary)|false|helm Asset|
|» docker.tag|body|string|false|docker Tag|
|» docker.asset|body|string(binary)|false|docker Asset|

<h3 id="uploadcomponent-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to upload a component|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Parameter 'repository' is required|None|

<aside class="success">
This operation does not require authentication
</aside>

## getComponentById

<a id="opIdgetComponentById"></a>

`GET /v1/components/{id}`

*Get a single component*

<h3 id="getcomponentbyid-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|string|true|ID of the component to retrieve|

> Example responses

> 200 Response

```json
{
  "id": "string",
  "repository": "string",
  "format": "string",
  "group": "string",
  "name": "string",
  "version": "string",
  "assets": [
    {
      "downloadUrl": "string",
      "path": "string",
      "id": "string",
      "repository": "string",
      "format": "string",
      "checksum": {
        "property1": "string",
        "property2": "string"
      },
      "contentType": "string",
      "lastModified": "2019-08-24T14:15:22Z",
      "lastDownloaded": "2019-08-24T14:15:22Z",
      "uploader": "string",
      "uploaderIp": "string",
      "fileSize": 0,
      "blobCreated": "2019-08-24T14:15:22Z",
      "blobStoreName": "string"
    }
  ]
}
```

<h3 id="getcomponentbyid-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|[ComponentXO](#schemacomponentxo)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to get component|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Component not found|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Malformed ID|None|

<aside class="success">
This operation does not require authentication
</aside>

## deleteComponent

<a id="opIddeleteComponent"></a>

`DELETE /v1/components/{id}`

*Delete a single component*

<h3 id="deletecomponent-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|string|true|ID of the component to delete|

<h3 id="deletecomponent-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|Component was successfully deleted|None|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions to delete component|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Component not found|None|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Malformed ID|None|

<aside class="success">
This operation does not require authentication
</aside>

<h1 id="nexus-repository-manager-rest-api-security-management-privileges">Security management: privileges</h1>

## getRepository

<a id="opIdgetRepository"></a>

`GET /v1/repositories/{repositoryName}`

*Get repository details*

<h3 id="getrepository-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|repositoryName|path|string|true|Name of the repository to get|

> Example responses

> 200 Response

```json
{
  "name": "string",
  "format": "string",
  "type": "string",
  "url": "string",
  "attributes": {
    "property1": {},
    "property2": {}
  }
}
```

<h3 id="getrepository-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|[RepositoryXO](#schemarepositoryxo)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication required|None|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Repository not found|None|

<aside class="success">
This operation does not require authentication
</aside>

## deleteRepository

<a id="opIddeleteRepository"></a>

`DELETE /v1/repositories/{repositoryName}`

*Delete repository of any format*

<h3 id="deleterepository-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|repositoryName|path|string|true|Name of the repository to delete|

<h3 id="deleterepository-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|Repository deleted|None|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication required|None|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Insufficient permissions|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Repository not found|None|

<aside class="success">
This operation does not require authentication
</aside>

## getRepositories

<a id="opIdgetRepositories"></a>

`GET /v1/repositories`

*List repositories*

> Example responses

> 200 Response

```json
[
  {
    "name": "string",
    "format": "string",
    "type": "string",
    "url": "string",
    "attributes": {
      "property1": {},
      "property2": {}
    }
  }
]
```

<h3 id="getrepositories-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|Inline|

<h3 id="getrepositories-responseschema">Response Schema</h3>

Status Code **200**

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[[RepositoryXO](#schemarepositoryxo)]|false|none|none|
|» name|string|false|none|none|
|» format|string|false|none|none|
|» type|string|false|none|none|
|» url|string|false|none|none|
|» attributes|object|false|none|none|
|»» **additionalProperties**|object|false|none|none|

<aside class="success">
This operation does not require authentication
</aside>

## searchAssets

<a id="opIdsearchAssets"></a>

`GET /v1/search/assets`

*Search assets*

All searches require at least one criterion of at least three characters before a trailing wildcard (\*) and cannot start with a wildcard (\*). Enclose your criteria in quotation marks to search an exact phrase; otherwise, search criteria will be split by any commas, spaces, dashes, or forward slashes.

<h3 id="searchassets-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|continuationToken|query|string|false|A token returned by a prior request. If present, the next page of results are returned|
|sort|query|string|false|The field to sort the results against, if left empty, a sort based on match weight will be used.|
|direction|query|string|false|The direction to sort records in, defaults to ascending ('asc') for all sort fields, except version, which defaults to descending ('desc')|
|timeout|query|integer(int32)|false|How long to wait for search results in seconds. If this value is not provided, the system default timeout will be used.|
|q|query|string|false|Query by keyword|
|repository|query|string|false|Repository name|
|format|query|string|false|Query by format|
|group|query|string|false|Component group|
|name|query|string|false|Component name|
|version|query|string|false|Component version|
|prerelease|query|string|false|Prerelease version flag|
|md5|query|string|false|Specific MD5 hash of component's asset|
|sha1|query|string|false|Specific SHA-1 hash of component's asset|
|sha256|query|string|false|Specific SHA-256 hash of component's asset|
|sha512|query|string|false|Specific SHA-512 hash of component's asset|
|composer.vendor|query|string|false|Vendor|
|composer.package|query|string|false|Package|
|composer.version|query|string|false|Version|
|conan.baseVersion|query|string|false|Conan base version|
|conan.channel|query|string|false|Conan channel|
|conan.revision|query|string|false|Conan recipe revision|
|conan.packageId|query|string|false|Conan package id|
|conan.packageRevision|query|string|false|Conan package revision|
|conan.baseVersion.strict|query|string|false|Conan base version strict|
|conan.revision.latest|query|string|false|Return latest revision|
|conan.settings.arch|query|string|false|Conan arch|
|conan.settings.os|query|string|false|Conan os|
|conan.settings.compiler|query|string|false|Conan compiler|
|conan.settings.compiler.version|query|string|false|Conan compiler version|
|conan.settings.compiler.runtime|query|string|false|Conan compiler runtime|
|docker.imageName|query|string|false|Docker image name|
|docker.imageTag|query|string|false|Docker image tag|
|docker.layerId|query|string|false|Docker layer ID|
|docker.contentDigest|query|string|false|Docker content digest|
|maven.groupId|query|string|false|Maven groupId|
|maven.artifactId|query|string|false|Maven artifactId|
|maven.baseVersion|query|string|false|Maven base version|
|maven.extension|query|string|false|Maven extension of component's asset|
|maven.classifier|query|string|false|Maven classifier of component's asset|
|gavec|query|string|false|Group asset version extension classifier|
|npm.scope|query|string|false|npm scope|
|npm.author|query|string|false|npm author|
|npm.description|query|string|false|npm description|
|npm.keywords|query|string|false|npm keywords|
|npm.license|query|string|false|npm license|
|npm.tagged_is|query|string|false|npm tagged is|
|npm.tagged_not|query|string|false|npm tagged not|
|nuget.id|query|string|false|NuGet id|
|nuget.tags|query|string|false|NuGet tags|
|nuget.title|query|string|false|NuGet title|
|nuget.authors|query|string|false|NuGet authors|
|nuget.description|query|string|false|NuGet description|
|nuget.summary|query|string|false|NuGet summary|
|nuget.is_prerelease|query|string|false|NuGet prerelease|
|p2.pluginName|query|string|false|p2 plugin name|
|pypi.classifiers|query|string|false|PyPI classifiers|
|pypi.description|query|string|false|PyPI description|
|pypi.keywords|query|string|false|PyPI keywords|
|pypi.summary|query|string|false|PyPI summary|
|rubygems.description|query|string|false|RubyGems description|
|rubygems.platform|query|string|false|RubyGems platform|
|rubygems.summary|query|string|false|RubyGems summary|
|tag|query|string|false|Component tag|
|yum.architecture|query|string|false|Yum architecture|
|yum.name|query|string|false|Yum package name|

#### Enumerated Values

|Parameter|Value|
|---|---|
|sort|group|
|sort|name|
|sort|version|
|sort|repository|
|direction|asc|
|direction|desc|

> Example responses

> 200 Response

```json
{
  "items": [
    {
      "downloadUrl": "string",
      "path": "string",
      "id": "string",
      "repository": "string",
      "format": "string",
      "checksum": {
        "property1": "string",
        "property2": "string"
      },
      "contentType": "string",
      "lastModified": "2019-08-24T14:15:22Z",
      "lastDownloaded": "2019-08-24T14:15:22Z",
      "uploader": "string",
      "uploaderIp": "string",
      "fileSize": 0,
      "blobCreated": "2019-08-24T14:15:22Z",
      "blobStoreName": "string"
    }
  ],
  "continuationToken": "string"
}
```

<h3 id="searchassets-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|successful operation|[PageAssetXO](#schemapageassetxo)|

<aside class="success">
This operation does not require authentication
</aside>

## searchAndDownloadAssets

<a id="opIdsearchAndDownloadAssets"></a>

`GET /v1/search/assets/download`

*Search and download asset*

Returns a 302 Found with location header field set to download URL. Unless a sort parameter is supplied, the search must return a single asset to receive download URL.

<h3 id="searchanddownloadassets-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|sort|query|string|false|The field to sort the results against, if left empty and more than 1 result is returned, the request will fail.|
|direction|query|string|false|The direction to sort records in, defaults to ascending ('asc') for all sort fields, except version, which defaults to descending ('desc')|
|timeout|query|integer(int32)|false|How long to wait for search results in seconds. If this value is not provided, the system default timeout will be used.|
|q|query|string|false|Query by keyword|
|repository|query|string|false|Repository name|
|format|query|string|false|Query by format|
|group|query|string|false|Component group|
|name|query|string|false|Component name|
|version|query|string|false|Component version|
|prerelease|query|string|false|Prerelease version flag|
|md5|query|string|false|Specific MD5 hash of component's asset|
|sha1|query|string|false|Specific SHA-1 hash of component's asset|
|sha256|query|string|false|Specific SHA-256 hash of component's asset|
|sha512|query|string|false|Specific SHA-512 hash of component's asset|
|composer.vendor|query|string|false|Vendor|
|composer.package|query|string|false|Package|
|composer.version|query|string|false|Version|
|conan.baseVersion|query|string|false|Conan base version|
|conan.channel|query|string|false|Conan channel|
|conan.revision|query|string|false|Conan recipe revision|
|conan.packageId|query|string|false|Conan package id|
|conan.packageRevision|query|string|false|Conan package revision|
|conan.baseVersion.strict|query|string|false|Conan base version strict|
|conan.revision.latest|query|string|false|Return latest revision|
|conan.settings.arch|query|string|false|Conan arch|
|conan.settings.os|query|string|false|Conan os|
|conan.settings.compiler|query|string|false|Conan compiler|
|conan.settings.compiler.version|query|string|false|Conan compiler version|
|conan.settings.compiler.runtime|query|string|false|Conan compiler runtime|
|docker.imageName|query|string|false|Docker image name|
|docker.imageTag|query|string|false|Docker image tag|
|docker.layerId|query|string|false|Docker layer ID|
|docker.contentDigest|query|string|false|Docker content digest|
|maven.groupId|query|string|false|Maven groupId|
|maven.artifactId|query|string|false|Maven artifactId|
|maven.baseVersion|query|string|false|Maven base version|
|maven.extension|query|string|false|Maven extension of component's asset|
|maven.classifier|query|string|false|Maven classifier of component's asset|
|gavec|query|string|false|Group asset version extension classifier|
|npm.scope|query|string|false|npm scope|
|npm.author|query|string|false|npm author|
|npm.description|query|string|false|npm description|
|npm.keywords|query|string|false|npm keywords|
|npm.license|query|string|false|npm license|
|npm.tagged_is|query|string|false|npm tagged is|
|npm.tagged_not|query|string|false|npm tagged not|
|nuget.id|query|string|false|NuGet id|
|nuget.tags|query|string|false|NuGet tags|
|nuget.title|query|string|false|NuGet title|
|nuget.authors|query|string|false|NuGet authors|
|nuget.description|query|string|false|NuGet description|
|nuget.summary|query|string|false|NuGet summary|
|nuget.is_prerelease|query|string|false|NuGet prerelease|
|p2.pluginName|query|string|false|p2 plugin name|
|pypi.classifiers|query|string|false|PyPI classifiers|
|pypi.description|query|string|false|PyPI description|
|pypi.keywords|query|string|false|PyPI keywords|
|pypi.summary|query|string|false|PyPI summary|
|rubygems.description|query|string|false|RubyGems description|
|rubygems.platform|query|string|false|RubyGems platform|
|rubygems.summary|query|string|false|RubyGems summary|
|tag|query|string|false|Component tag|
|yum.architecture|query|string|false|Yum architecture|
|yum.name|query|string|false|Yum package name|

#### Enumerated Values

|Parameter|Value|
|---|---|
|sort|group|
|sort|name|
|sort|version|
|sort|repository|
|direction|asc|
|direction|desc|

<h3 id="searchanddownloadassets-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|ValidationErrorXO{id='*', message='Search returned multiple assets, please refine search criteria to find a single asset or use the sort query parameter to retrieve the first result.'}|None|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Asset search returned no results|None|

<aside class="success">
This operation does not require authentication
</aside>
