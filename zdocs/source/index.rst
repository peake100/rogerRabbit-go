Islelib-Go
==========

.. toctree::
   :maxdepth: 2
   :caption: Contents:

``islelib`` is Illuscio's python library template. To build your own documentation,
simply start using it. Below we will show some example documentation with the basic
functions of this library template.

Table of Contents
=================

* :ref:`basic-usage`
* :ref:`setting-up`
* :ref:`writing`
* :ref:`deploying`
* :ref:`qol`

.. _basic-usage:

Basic Usage
===========

   >>> import islelib
   [basic useage example goes here]

Islelib comes with a number of pre-built quality-of-life macros for developers so they
can code more and manage less, most of which are accessed through ``make`` commands.

In addition to your lib's package folder, islelib has two other main directories:

   * ``./zdocs`` - where docs are built to and stored
   * ``./zdevelop`` - where tests, maintenance scripts, and other information is stored

.. note::

    Islelib separates tests into their own zdevelop/tests directory. Because of the way
    go works, this only allows the importing of public APIs, so unit testing of private
    functions or method will still need to live alongside the regular files.

    The philosophy behind this is that most tests will be tests of the public APIs, and
    it leads to a cleaner repository to have tests tucked away in their own folder.

In addition, the following files are used:

   * ``./README.md`` - brief description and pointer to doc link for `Github`_.
   * ``./setup.cfg`` - when possible, settings for all tools are stored here.
   * ``./setup.py`` - required for sphinx build command.
   * ``./revive.toml`` - configuration for linting with `Revive`_.
   * ``./Makefile`` - contains make commands for the development features detailed in
     this doc.
   * ``./azure_pipelines.yml`` - build process definition for `Azure Pipelines`_.

.. _setting-up:

Setting up your Library
=======================

Getting started is easy, just follow the below steps. Many of these steps include
``Make`` scripts that help you get up and running quickly. To run the ``Make`` commands,
ensure that the active directory of your terminal session is ``"islelib-py"``

1. Clone islelib-go from Github
--------------------------------

navigate to where you wish to keep your project in terminal: ::

   >>> cd /path/to/local_repository
   >>> git clone git@github.com:bpeake-illuscio/islelib-go.git

once the library is cloned, move into it as your active directory: ::

    >>> cd islelib-go

2. Pick a Name
--------------

Illuscio uses the ''isle'' prefix convention (phonetically sounds like I-L/"Aye-EL" as
opposed to "ill". Examples include ``isle_type``, ``isle_collections``, etc.

When you have chosen a name for your new lib, simply type: ::

   >>> make name n=libname
   library renamed! to switch your current directory, use the following	command:
   cd '/path/to/libname-py'

... where ``libname`` is the name of your new library. This will:

   * change the name of any packages with an __init__ to ``libname`` (uses a find and replace from the old name when applicable).
   * change all of the relevant setup.cfg options to ``libname``
   * change the top level folder to ``libname-py``
   * remove old ``islelib.egg`` folder

3. Pick a Description
---------------------

In the ``./setup.cfg`` file, under the ``[metadata]`` header, change the ``description``
field to a brief description of your project.

4. Create a Virtual Environment
-------------------------------

Although this is a go library, python is used for creating higher-level documentation
through sphinx, allowing the creation of high-level tutorial docs. See below for more
information.

To set up a virtual environment through virtualenv, type: ::

   >>> make venv

This will install a new virtual environment at ``~/venvs/[libname]-go-[## python version]``.

Example name: ``libname-go-37``

By default, this command uses your current "python3" alias, but a different version
can be supplied with a `py=` option: ::

   >>> make venv py="/Library/Frameworks/Python.framework/Versions/3.7/bin/python3"
   venv created! To enter virtual env, run :
   . ~/.bash_profile
   then run:
   env_go-libname-37

``make venv`` also registers the environment and library directory to your ~/.bash_profile.
This allows you to easily enter a development environment in terminal by typing: ::

   >>> env_go-libname-37

... where `libname` is the name of your lib and `37` is the python version of the venv.
This command is equivalent to: ::

   >>> cd /path/to/libname-py
   >>> source ~/venvs/libname-go-37/bin/activate

In order to use the new alias, you will need to refresh your current terminal session by
typing: ::

   >>> . ~/.bash_profile

5. Install the Dev Environment
------------------------------

islelib already comes pre-built with all the options and tools needed to write a generic
library. To install these tools into a python environment, type: ::

   >>> make install-dev

These tools include automation for building, testing and docing your new
library.

You will need to have Make installed on your machine. For OSX, you will be prompted
to install make through XCode when you attempt to run this command if it is not
already installed.

6. Initialize a new git repo
----------------------------

You should delete the existing ``.git`` folder for the repository, then initialize a
clean repo by typing: ::

   >>> git init

In the future, you may wish to cherry-pick commits / updates to this template into
your own libraries. A guide for how to do that can be found here:

[Guide needs to be written]

7. Register your library
------------------------

Please reference the relevant documentation for registering your library in Github,
Readthedocs, Azure Pipelines, etc. Links to relevant guides can be found below:

[Guides need to be written]

.. _writing:

Writing Your Library
====================

1. Style
--------

Illuscio's style guide is simple and straightforward:

   1. `Gofmt`_ first
   2. `Revive`_ second
   3. When 1 & 2 contradict: see 1

2. Lint
-------

To check the formatting of your library, type: ::

   >>> make lint

This will run the following tools to tell you where adjustments need to be made:

   * `Revive`_

`Revive`_ will check your formatting and report any instances where
code does not conform to it's standards. The configuration file for which rules revive
enforces is found in: ::

    ./revive.toml

These lint checks are also performed during deployment, and will cause failed code to
be kept from deploying to production.

3. Re-format
------------

Strict pep8 and Black adherence, while useful in many ways to the organization, can be
annoying and distracting to individual engineers. To help with this, the islelib
template comes with tools to re-format your code for you.

To re-format your code, type: ::

   >>> make format

This will run the following tools:

   * `gofmt`_

With these tools, keeping your code properly formatted is minimally invasive, and as an
organization will lead to a more consistent, maintainable codebase.

4. Test
-------

Tests are placed in ``zdevelop/tests``, and use the `testing`_ library. To run your
tests type: ::

   >>> make test

... and watch the magic happen. This macro also creates coverage and error reports.
Coverage reports show what percentage of each file's code is tested. These reports can
be found in the following locations, and will be automatically opened in your default
browser once the tests complete:

   * results: ``zdevelop/tests/_reports/test_report.html``
   * coverage: ``zdevelop/tests/_reports/coverage/coverage.out``

Illuscio requires >= 85% code coverage in all files to publish a library. Libraries
with less than 85% coverage in any given file will be kicked back or will need to have
an exception granted.

Likewise, code will be tested upon deployment and kicked back in the case of failures.
The brief example tests in this library includes a failed test.

.. note::

    This template contains a testing dependency for `testify`_, which is suggested for
    making asserts and setup / teardown groups.

5. Document
-----------

islelib uses a combination of `GoDoc`_ and `Sphinx`_ to create it's documentation which
enables the creation of high-level docs / tutorials using (which godoc is not well
uited for) .rst files and `Sphinx`_, which can then link to traditional API docs created
using `GoDoc`_.

High level documentation's entry point is: ::

    zdocs/source/index.rst

Which is where the text you are currently reading is located.

During the build process, API documentation is automatically generated from `GoDoc`_
and included as static files. In order to link to this generated API docs within the
high level docs, type the following: ::

    API documentation is created using godoc and can be
    `found here <_static/godoc-root.html>`_.

Example: API documentation is created using godoc and can be
`found here <_static/godoc-root.html>`_.

To build docs for your new library, type: ::

   >>> make doc

Docs will be generated at ``./zdocs/_build/index.html``. This command will also open
the newly created documentation in your default browser.

.. _deploying:

Deploying Your Library
======================

1. Make Commits:
----------------
Make your commits as you work. Your commits will be made to the ``dev`` branch, changes
are pushed to master automatically once builds are passed.

2. Version:
-----------

The major / minor version of the library are set in the ``setup.cfg`` file under
``version:target``.

Patch versions are generated automatically by the build system. So if ``version:target``
is ``1.2`` and the last published build was ``1.2.4`` the next build created will
become ``1.2.5``.

When a new major / minor version bump is desired, simply change the ``version:target``
value, ie ``1.3`` or ``2.0``.

3. Push:
--------

When you are ready, push your code to github. This will set off a chain of events that
will:

   * automatically run formatting and unit tests
   * if tests are passed, build and push your library to be available to other developers
   * builds and published documentation to `Amazon S3`_

4. Build:
---------

islelib uses `Azure Pipelines`_ to automatically run builds.

For more information on azure builds, see the `azure build templates repo`_.

.. _qol:

Other Quality of Life Development Functions
===========================================

1. Clean Caches
---------------

``make clean`` will clear the following files from your library:

   * pytest cache
   * mypy cache
   * .coverage cache
   * ./build directory
   * ./dist directory
   * all .pyc files in the active directory tree
   * the ``build`` folder in ``./zdevelop/docs``
   * the ``.idea`` folder generated by pycharm to reset pycharm's cache


2. Scratch Folder
-----------------

The folder ``zdevelop/scratch`` is included in .gitignore, so you can store scratch work
to do quick tests in this directory without accidentally causing a commit conflict.


.. web links:
.. _Revive: https://github.com/mgechev/revive
.. _testing: https://golang.org/pkg/testing/
.. _testify: https://github.com/stretchr/testify
.. _Github: https://github.com/
.. _GoDoc: https://godoc.org/golang.org/x/tools/cmd/godoc
.. _Gofmt: https://golang.org/cmd/gofmt/
.. _pytest: https://docs.pytest.org/en/latest/
.. _Sphinx: http://www.sphinx-doc.org/en/master/
.. _Azure Pipelines: https://azure.microsoft.com/en-us/services/devops/pipelines/
.. _PyPri: https://www.python-private-package-index.com/
.. _Azure Artifacts: https://azure.microsoft.com/en-us/services/devops/artifacts/
.. _Python Azure Artifacts Feed: https://dev.azure.com/illuscio/Python%20Packages/_packaging?_a=feed&feed=isle_pypi_libs
.. _Pipeline dashboard:: https://dev.azure.com/illuscio/Python%20Packages/_build?definitionId=1
.. _twine: https://twine.readthedocs.io/en/latest/
.. _Amazon S3: https://aws.amazon.com/s3/
.. _Cloudfront: https://aws.amazon.com/cloudfront/
.. _this lambda edge template: https://console.aws.amazon.com/lambda/home?region=us-east-1#/create/app?applicationId=arn:aws:serverlessrepo:us-east-1:520945424137:applications/cloudfront-authorization-at-edge
.. _azure build templates repo: https://github.com/illuscio-dev/azure-pipelines-templates
